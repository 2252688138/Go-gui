// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package gpu

const vertShaderShape = `#version 330 core
layout(location=0) in vec2 aPos;
layout(location=1) in vec2 aLocal;
layout(location=2) in vec4 aColor;
layout(location=3) in vec4 aColor2;
layout(location=4) in vec2 aHalf;
layout(location=5) in vec4 aRadius;
layout(location=6) in vec4 aParams;

uniform vec2 uViewport;

out vec2 vLocal;
out vec4 vColor;
out vec4 vColor2;
out vec2 vHalf;
out vec4 vRadius;
out vec4 vParams;
out vec2 vT;

void main() {
    gl_Position = vec4((aPos / uViewport) * 2.0 - 1.0, 0.0, 1.0);
    gl_Position.y = -gl_Position.y;
    vLocal = aLocal;
    vColor = aColor;
    vColor2 = aColor2;
    vHalf = aHalf;
    vRadius = aRadius;
    vParams = aParams;
    vec2 t = (aLocal + aHalf) / max(aHalf * 2.0, vec2(1.0));
    vT = clamp(t, 0.0, 1.0);
}
`

const fragShaderShape = `#version 330 core
in vec2 vLocal;
in vec4 vColor;
in vec4 vColor2;
in vec2 vHalf;
in vec4 vRadius;
in vec4 vParams;
in vec2 vT;

out vec4 fragColor;

float sdRoundBox(vec2 p, vec2 b, vec4 r) {
    r.xy = (p.x > 0.0) ? r.xy : r.zw;
    r.x  = (p.y > 0.0) ? r.x  : r.y;
    vec2 q = abs(p) - b + r.x;
    return min(max(q.x, q.y), 0.0) + length(max(q, 0.0)) - r.x;
}

void main() {
    float mode     = vParams.y;   // 0=fill 1=stroke 2=shadow
    // 下限 0.25：超采样时 CPU 侧会把过渡带按 1/ss 收紧（设备像素恒定），
    // 原先 0.5 的下限会把 2× 档收紧后的 0.35 又抬回 0.5，锐度被吃掉一半。
    float feather  = max(vParams.x, 0.25);
    float strokeW  = vParams.z;
    float glow     = vParams.w;

    float d = sdRoundBox(vLocal, vHalf, vRadius);
    // SDF 模糊：向外扩散 glow 像素，高斯近似衰减
    if (glow > 0.0) {
        float k = clamp(-d / max(glow, 0.001), 0.0, 1.0);
        d = d + glow * (1.0 - k);
    }

    float alpha;
    if (mode < 0.5) {
        alpha = 1.0 - smoothstep(-feather, feather, d);
    } else if (mode < 1.5) {
        alpha = 1.0 - smoothstep(-feather, feather, abs(d + strokeW * 0.5) - strokeW * 0.5);
    } else {
        alpha = (1.0 - smoothstep(-glow, glow, d)) * 0.9;
    }

    if (alpha <= 0.001) discard;

    vec4 col = vColor;
    if (vColor2.a >= 0.0) {
        col = mix(vColor, vColor2, vT.x);
    }
    fragColor = vec4(col.rgb, col.a * alpha);
}
`

const vertShaderText = `#version 330 core
layout(location=0) in vec2 aPos;
layout(location=1) in vec2 aUV;
layout(location=2) in vec4 aColor;
layout(location=3) in float aMode;

uniform vec2 uViewport;

out vec2 vUV;
out vec4 vColor;
out float vMode;

void main() {
    gl_Position = vec4((aPos / uViewport) * 2.0 - 1.0, 0.0, 1.0);
    gl_Position.y = -gl_Position.y;
    vUV = aUV;
    vColor = aColor;
    vMode = aMode;
}
`

const fragShaderText = `#version 330 core
in vec2 vUV;
in vec4 vColor;
in float vMode;
out vec4 fragColor;

uniform sampler2D uAtlas;
uniform float uTextGamma;

void main() {
    vec4 c = texture(uAtlas, vUV);
    if (vMode > 0.5) {
        // 彩色字形（Emoji）：直接输出纹理颜色
        if (c.a <= 0.004) discard;
        fragColor = vec4(c.rgb, c.a * vColor.a);
        return;
    }
    float a = c.r;
    if (a <= 0.004) discard;
    // 灰度覆盖率 gamma：浏览器（Skia/DirectWrite）在 sRGB 空间混合前会做 gamma 校正，
    // 等效于把中间覆盖率拉高，小字号笔画因此更「实」。这里只对「亮字」（暗底浅字）补偿：
    // 暗字浅底时线性混合反而更淡，我们当前的 sRGB 直混已经比浏览器更粗，不需再拉。
    // uTextGamma = 1.0 时完全关闭。
    float lum = dot(vColor.rgb, vec3(0.2126, 0.7152, 0.0722));
    if (uTextGamma != 1.0 && lum > 0.5) {
        a = pow(a, uTextGamma);
    }
    fragColor = vec4(vColor.rgb, vColor.a * a);
}
`

// 毛玻璃：双 Kawase 采样（降采样 + 上采样共用）
const vertShaderBlur = `#version 330 core
layout(location=0) in vec2 aPos;
layout(location=1) in vec2 aUV;

out vec2 vUV;

void main() {
    gl_Position = vec4(aPos, 0.0, 1.0);
    vUV = aUV;
}
`

const fragShaderBlur = `#version 330 core
in vec2 vUV;
out vec4 fragColor;

uniform sampler2D uTex;
uniform vec2 uTexel;
uniform float uRadius;

void main() {
    vec2 o = uTexel * uRadius;
    vec4 c = texture(uTex, vUV) * 4.0;
    c += texture(uTex, vUV + vec2(-o.x, 0.0)) * 2.0;
    c += texture(uTex, vUV + vec2( o.x, 0.0)) * 2.0;
    c += texture(uTex, vUV + vec2(0.0, -o.y)) * 2.0;
    c += texture(uTex, vUV + vec2(0.0,  o.y)) * 2.0;
    c += texture(uTex, vUV + vec2(-o.x, -o.y));
    c += texture(uTex, vUV + vec2( o.x, -o.y));
    c += texture(uTex, vUV + vec2(-o.x,  o.y));
    c += texture(uTex, vUV + vec2( o.x,  o.y));
    fragColor = c / 16.0;
}
`

// 超采样降采样（resolve）：把 ss 倍分辨率的离屏目标平均回窗口像素。
// 每个输出像素取 4×4 个子带中心样本，覆盖它对应的 span×span 个源像素，
// 等价于对整帧做 box 滤波——圆角/描边/字形都因此获得亚像素级覆盖率。
const fragShaderResolve = `#version 330 core
in vec2 vUV;
out vec4 fragColor;

uniform sampler2D uTex;
uniform vec2 uTexel;  // 1 / 超采样目标尺寸
uniform float uSpan;  // 每个输出像素覆盖的源像素数（= 超采样倍数）
uniform float uSharpen; // 降采样后的轻度锐化强度（0 = 关闭）

void main() {
    vec2 base = uTexel * (uSpan * 0.125);
    vec4 c = vec4(0.0);
    for (int y = 0; y < 4; y++) {
        for (int x = 0; x < 4; x++) {
            vec2 o = vec2(float(x) * 2.0 - 3.0, float(y) * 2.0 - 3.0) * base;
            c += texture(uTex, vUV + o);
        }
    }
    c *= 0.0625;
    // 轻度 unsharp：box 平均会把 1px 级细节（横竖笔、1px 线条）压低成「软」观感，
    // 人眼会误判为「还不如 1:1 清晰」。以 1 输出像素半径的十字平均作为模糊基准，
    // 只补回被平滑掉的一小部分成分：既保留抗锯齿，又不至于出现白色光晕。
    if (uSharpen > 0.0) {
        vec2 o = uTexel * uSpan;
        vec4 blur = (texture(uTex, vUV + vec2(o.x, 0.0)) +
                     texture(uTex, vUV - vec2(o.x, 0.0)) +
                     texture(uTex, vUV + vec2(0.0, o.y)) +
                     texture(uTex, vUV - vec2(0.0, o.y))) * 0.25;
        c = clamp(c + (c - blur) * uSharpen, 0.0, 1.0);
    }
    fragColor = c;
}
`

const vertShaderImg = `#version 330 core
layout(location=0) in vec2 aPos;
layout(location=1) in vec2 aUV;
layout(location=2) in vec4 aColor;
layout(location=3) in vec2 aLocal;
layout(location=4) in vec2 aHalf;
layout(location=5) in vec4 aRadius;

uniform vec2 uViewport;

out vec2 vUV;
out vec4 vColor;
out vec2 vLocal;
out vec2 vHalf;
out vec4 vRadius;

void main() {
    gl_Position = vec4((aPos / uViewport) * 2.0 - 1.0, 0.0, 1.0);
    gl_Position.y = -gl_Position.y;
    vUV = aUV;
    vColor = aColor;
    vLocal = aLocal;
    vHalf = aHalf;
    vRadius = aRadius;
}
`

const fragShaderImg = `#version 330 core
in vec2 vUV;
in vec4 vColor;
in vec2 vLocal;
in vec2 vHalf;
in vec4 vRadius;
out vec4 fragColor;

uniform sampler2D uTex;

float sdRoundBox(vec2 p, vec2 b, vec4 r) {
    r.xy = (p.x > 0.0) ? r.xy : r.zw;
    r.x  = (p.y > 0.0) ? r.x  : r.y;
    vec2 q = abs(p) - b + r.x;
    return min(max(q.x, q.y), 0.0) + length(max(q, 0.0)) - r.x;
}

void main() {
    float d = sdRoundBox(vLocal, vHalf, vRadius);
    float a = 1.0 - smoothstep(-0.7, 0.7, d);
    if (a <= 0.002) discard;
    vec4 c = texture(uTex, vUV);
    fragColor = vec4(c.rgb * vColor.rgb, c.a * vColor.a * a);
}
`
