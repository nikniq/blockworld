package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Post-processing: the 3D scene renders into a texture with a sampleable
// depth buffer and is drawn to the screen through one pass that inks
// outlines where depth breaks or creases (the sketched look), anti-aliases
// with FXAA, and adds a gentle vignette.

const postVertex = `#version 330
in vec3 vertexPosition;
in vec2 vertexTexCoord;
in vec4 vertexColor;
uniform mat4 mvp;
out vec2 fragTexCoord;
out vec4 fragColor;
void main() {
    fragTexCoord = vertexTexCoord;
    fragColor = vertexColor;
    gl_Position = mvp * vec4(vertexPosition, 1.0);
}`

const postFragment = `#version 330
in vec2 fragTexCoord;
in vec4 fragColor;
uniform sampler2D texture0;
uniform sampler2D depthTex;
uniform vec4 colDiffuse;
uniform vec2 resolution;
uniform vec2 clip;        // near and far plane distances
uniform float ink;        // 0 off, 1 outlines on
uniform float aa;         // 0 off, 1 FXAA on
out vec4 finalColor;

#define FXAA_SPAN_MAX 8.0
#define FXAA_REDUCE_MUL (1.0 / 8.0)
#define FXAA_REDUCE_MIN (1.0 / 128.0)

// linear depth in world units from the non-linear depth buffer
float ldepth(vec2 uv) {
    float d = texture(depthTex, uv).r * 2.0 - 1.0;
    return 2.0 * clip.x * clip.y / (clip.y + clip.x - d * (clip.y - clip.x));
}

void main() {
    vec2 inv = 1.0 / resolution;
    vec3 luma = vec3(0.299, 0.587, 0.114);
    vec3 rgb = texture(texture0, fragTexCoord).rgb;
    if (aa > 0.5) {
        vec3 rgbNW = texture(texture0, fragTexCoord + vec2(-1.0, -1.0) * inv).rgb;
        vec3 rgbNE = texture(texture0, fragTexCoord + vec2(1.0, -1.0) * inv).rgb;
        vec3 rgbSW = texture(texture0, fragTexCoord + vec2(-1.0, 1.0) * inv).rgb;
        vec3 rgbSE = texture(texture0, fragTexCoord + vec2(1.0, 1.0) * inv).rgb;
        vec3 rgbM  = rgb;
        float lumaNW = dot(rgbNW, luma);
        float lumaNE = dot(rgbNE, luma);
        float lumaSW = dot(rgbSW, luma);
        float lumaSE = dot(rgbSE, luma);
        float lumaM  = dot(rgbM, luma);
        float lumaMin = min(lumaM, min(min(lumaNW, lumaNE), min(lumaSW, lumaSE)));
        float lumaMax = max(lumaM, max(max(lumaNW, lumaNE), max(lumaSW, lumaSE)));
        vec2 dir;
        dir.x = -((lumaNW + lumaNE) - (lumaSW + lumaSE));
        dir.y =  ((lumaNW + lumaSW) - (lumaNE + lumaSE));
        float dirReduce = max((lumaNW + lumaNE + lumaSW + lumaSE) * (0.25 * FXAA_REDUCE_MUL), FXAA_REDUCE_MIN);
        float rcpDirMin = 1.0 / (min(abs(dir.x), abs(dir.y)) + dirReduce);
        dir = min(vec2(FXAA_SPAN_MAX), max(vec2(-FXAA_SPAN_MAX), dir * rcpDirMin)) * inv;
        vec3 rgbA = 0.5 * (texture(texture0, fragTexCoord + dir * (1.0 / 3.0 - 0.5)).rgb +
                           texture(texture0, fragTexCoord + dir * (2.0 / 3.0 - 0.5)).rgb);
        vec3 rgbB = rgbA * 0.5 + 0.25 * (texture(texture0, fragTexCoord + dir * -0.5).rgb +
                                         texture(texture0, fragTexCoord + dir * 0.5).rgb);
        float lumaB = dot(rgbB, luma);
        rgb = (lumaB < lumaMin || lumaB > lumaMax) ? rgbA : rgbB;
    }
    if (ink > 0.5) {
        // Line weight scales with the framebuffer so HiDPI screens get the same look.
        float px = max(1.25, resolution.y / 720.0);
        vec2 o = inv * px;
        float z  = ldepth(fragTexCoord);
        float zl = ldepth(fragTexCoord - vec2(o.x, 0.0));
        float zr = ldepth(fragTexCoord + vec2(o.x, 0.0));
        float zu = ldepth(fragTexCoord - vec2(0.0, o.y));
        float zd = ldepth(fragTexCoord + vec2(0.0, o.y));
        // Silhouettes: a jump in depth between neighbours, relative to distance.
        float jump = max(max(abs(zl - z), abs(zr - z)), max(abs(zu - z), abs(zd - z)));
        float sil = smoothstep(0.012 * z + 0.02, 0.03 * z + 0.08, jump);
        // Creases: the depth slope changes, as where a block top meets its side.
        float crease = abs(zl + zr - 2.0 * z) + abs(zu + zd - 2.0 * z);
        float cr = smoothstep(0.0025 * z + 0.003, 0.009 * z + 0.012, crease);
        float edge = max(sil, cr * 0.8);
        // Lines fade with distance so the far hills are not a black scribble.
        edge *= 1.0 - smoothstep(60.0, 160.0, z);
        vec3 inkColor = vec3(0.07, 0.05, 0.09);
        rgb = mix(rgb, inkColor, edge * 0.85);
    }
    // Vignette.
    vec2 d = fragTexCoord - 0.5;
    rgb *= 1.0 - 0.35 * dot(d, d) * 2.0;
    finalColor = vec4(rgb, 1.0) * colDiffuse;
}`

type PostFX struct {
	target   rl.RenderTexture2D
	shader   rl.Shader
	locRes   int32
	locDepth int32
	locClip  int32
	locInk   int32
	locAA    int32
	ok       bool
	w, h     int32
}

var postfx PostFX

// loadTarget builds a framebuffer whose depth attachment is a texture the
// outline pass can read (raylib's own render textures use a renderbuffer).
func loadTarget(w, h int32) (rl.RenderTexture2D, bool) {
	var t rl.RenderTexture2D
	blank := rl.GenImageColor(int(w), int(h), rl.Blank)
	t.Texture = rl.LoadTextureFromImage(blank)
	rl.UnloadImage(blank)
	t.Depth = rl.Texture2D{ID: rl.LoadTextureDepth(w, h, false), Width: w, Height: h, Mipmaps: 1, Format: 19}
	t.ID = rl.LoadFramebuffer()
	rl.EnableFramebuffer(t.ID)
	rl.FramebufferAttach(t.ID, t.Texture.ID, rl.AttachmentColorChannel0, rl.AttachmentTexture2d, 0)
	rl.FramebufferAttach(t.ID, t.Depth.ID, rl.AttachmentDepth, rl.AttachmentTexture2d, 0)
	ok := rl.FramebufferComplete(t.ID)
	rl.DisableFramebuffer()
	if !ok {
		rl.UnloadRenderTexture(t)
		return rl.LoadRenderTexture(w, h), false
	}
	return t, true
}

// begin starts rendering the scene into the offscreen target (recreated on resize).
func (p *PostFX) begin() bool {
	// The target matches the framebuffer, which is larger than the logical window on scaled displays.
	scale := quality().RenderScale
	sw, sh := int32(float32(rl.GetRenderWidth())*scale), int32(float32(rl.GetRenderHeight())*scale)
	if sw <= 0 || sh <= 0 {
		return false
	}
	if !p.ok {
		p.shader = rl.LoadShaderFromMemory(postVertex, postFragment)
		if !rl.IsShaderValid(p.shader) {
			return false
		}
		p.locRes = rl.GetShaderLocation(p.shader, "resolution")
		p.locDepth = rl.GetShaderLocation(p.shader, "depthTex")
		p.locClip = rl.GetShaderLocation(p.shader, "clip")
		p.locInk = rl.GetShaderLocation(p.shader, "ink")
		p.locAA = rl.GetShaderLocation(p.shader, "aa")
		p.ok = true
	}
	if p.w != sw || p.h != sh {
		if p.w != 0 {
			rl.UnloadRenderTexture(p.target)
		}
		var depthOK bool
		p.target, depthOK = loadTarget(sw, sh)
		if !depthOK {
			p.target.Depth.ID = 0
		}
		rl.SetTextureFilter(p.target.Texture, rl.FilterBilinear)
		p.w, p.h = sw, sh
	}
	rl.BeginTextureMode(p.target)
	return true
}

// end finishes the scene and composites it to the screen. The target is
// framebuffer-sized; raylib's EndTextureMode restores the DPI scale, so the
// blit is expressed in logical window coordinates.
func (p *PostFX) end() {
	rl.EndTextureMode()
	rl.SetShaderValue(p.shader, p.locRes, []float32{float32(p.w), float32(p.h)}, rl.ShaderUniformVec2)
	rl.SetShaderValue(p.shader, p.locClip, []float32{float32(rl.GetCullDistanceNear()), float32(rl.GetCullDistanceFar())}, rl.ShaderUniformVec2)
	ink, aa := float32(0), float32(0)
	if settings.Ink && p.target.Depth.ID != 0 {
		ink = 1
	}
	if settings.Antialias {
		aa = 1
	}
	rl.SetShaderValue(p.shader, p.locInk, []float32{ink}, rl.ShaderUniformFloat)
	rl.SetShaderValue(p.shader, p.locAA, []float32{aa}, rl.ShaderUniformFloat)
	rl.BeginShaderMode(p.shader)
	if ink > 0 {
		rl.SetShaderValueTexture(p.shader, p.locDepth, p.target.Depth)
	}
	src := rl.NewRectangle(0, 0, float32(p.w), -float32(p.h))
	dst := rl.NewRectangle(0, 0, float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight()))
	rl.DrawTexturePro(p.target.Texture, src, dst, rl.Vector2{}, 0, rl.White)
	rl.EndShaderMode()
}
