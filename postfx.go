package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Post-processing: the 3D scene renders into a texture and is drawn to the
// screen through an FXAA pass (anti-aliasing) with a gentle vignette.

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
uniform vec4 colDiffuse;
uniform vec2 resolution;
out vec4 finalColor;

#define FXAA_SPAN_MAX 8.0
#define FXAA_REDUCE_MUL (1.0 / 8.0)
#define FXAA_REDUCE_MIN (1.0 / 128.0)

void main() {
    vec2 inv = 1.0 / resolution;
    vec3 rgbNW = texture(texture0, fragTexCoord + vec2(-1.0, -1.0) * inv).rgb;
    vec3 rgbNE = texture(texture0, fragTexCoord + vec2(1.0, -1.0) * inv).rgb;
    vec3 rgbSW = texture(texture0, fragTexCoord + vec2(-1.0, 1.0) * inv).rgb;
    vec3 rgbSE = texture(texture0, fragTexCoord + vec2(1.0, 1.0) * inv).rgb;
    vec3 rgbM  = texture(texture0, fragTexCoord).rgb;
    vec3 luma = vec3(0.299, 0.587, 0.114);
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
    vec3 rgb = (lumaB < lumaMin || lumaB > lumaMax) ? rgbA : rgbB;
    // Vignette.
    vec2 d = fragTexCoord - 0.5;
    rgb *= 1.0 - 0.35 * dot(d, d) * 2.0;
    finalColor = vec4(rgb, 1.0) * colDiffuse;
}`

type PostFX struct {
	target rl.RenderTexture2D
	shader rl.Shader
	locRes int32
	ok     bool
	w, h   int32
}

var postfx PostFX

// begin starts rendering the scene into the offscreen target (recreated on resize).
func (p *PostFX) begin() bool {
	sw, sh := int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight())
	if !p.ok {
		p.shader = rl.LoadShaderFromMemory(postVertex, postFragment)
		if !rl.IsShaderValid(p.shader) {
			return false
		}
		p.locRes = rl.GetShaderLocation(p.shader, "resolution")
		p.ok = true
	}
	if p.w != sw || p.h != sh {
		if p.w != 0 {
			rl.UnloadRenderTexture(p.target)
		}
		p.target = rl.LoadRenderTexture(sw, sh)
		rl.SetTextureFilter(p.target.Texture, rl.FilterBilinear)
		p.w, p.h = sw, sh
	}
	rl.BeginTextureMode(p.target)
	return true
}

// end finishes the scene and composites it to the screen with FXAA.
func (p *PostFX) end() {
	rl.EndTextureMode()
	rl.SetShaderValue(p.shader, p.locRes, []float32{float32(p.w), float32(p.h)}, rl.ShaderUniformVec2)
	rl.BeginShaderMode(p.shader)
	src := rl.NewRectangle(0, 0, float32(p.w), -float32(p.h))
	dst := rl.NewRectangle(0, 0, float32(p.w), float32(p.h))
	rl.DrawTexturePro(p.target.Texture, src, dst, rl.Vector2{}, 0, rl.White)
	rl.EndShaderMode()
}
