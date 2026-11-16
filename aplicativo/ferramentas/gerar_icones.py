#!/usr/bin/env python3
"""Gera os ícones do launcher Android e a imagem da tela de login a partir da logo.

Fonte única: assets/logo/logo.svg (tela de 108 x 108, a mesma do ícone
adaptativo). O retângulo de id "fundo" é removido para obter a camada de
frente; o fundo do ícone adaptativo é uma cor sólida (@color/ic_launcher_background).

Saídas (caminhos relativos a aplicativo/):
  android/app/src/main/res/mipmap-*/ic_launcher_foreground.png  camada de frente, 108 dp
  android/app/src/main/res/mipmap-*/ic_launcher.png             ícone legado quadrado, 48 dp
  android/app/src/main/res/mipmap-*/ic_launcher_round.png       ícone legado redondo, 48 dp
  android/app/src/main/res/mipmap-anydpi-v26/ic_launcher*.xml   ícone adaptativo
  android/app/src/main/res/values/ic_launcher_background.xml    cor do fundo
  assets/logo/logo.png, 2.0x/logo.png, 3.0x/logo.png           marca da tela de login, 64 px lógicos

Renderização com librsvg e cairo pelo PyGObject, sem dependência no
pubspec. No Ubuntu/Debian: apt install python3-gi python3-cairo
gir1.2-rsvg-2.0. A prévia (--previa) usa também o Pillow.

Uso (de qualquer diretório):
  python3 aplicativo/ferramentas/gerar_icones.py [--previa arquivo.png]
"""

from __future__ import annotations

import argparse
import math
import xml.etree.ElementTree as ET
from pathlib import Path

import cairo
import gi

gi.require_version("Rsvg", "2.0")
from gi.repository import Rsvg  # noqa: E402

APLICATIVO = Path(__file__).resolve().parent.parent
LOGO = APLICATIVO / "assets" / "logo" / "logo.svg"
RES = APLICATIVO / "android" / "app" / "src" / "main" / "res"

COR_FUNDO = "#101418"  # scaffoldBackgroundColor de lib/app/tema.dart
LADO = 108.0  # tela do ícone adaptativo, em dp
VISIVEL = 72.0  # parte da tela que o launcher mostra
RAIO_SEGURO = 33.0  # zona segura: círculo de 66 dp de diâmetro
LOGIN_DP = 64  # mesmo tamanho do ícone que a logo substitui na tela de login

DENSIDADES = {"mdpi": 1.0, "hdpi": 1.5, "xhdpi": 2.0, "xxhdpi": 3.0, "xxxhdpi": 4.0}

SVG_NS = "http://www.w3.org/2000/svg"
ET.register_namespace("", SVG_NS)


def carregar(sem_fundo: bool) -> Rsvg.Handle:
    arvore = ET.parse(LOGO)
    raiz = arvore.getroot()
    if sem_fundo:
        fundo = raiz.find(f"{{{SVG_NS}}}*[@id='fundo']")
        if fundo is None:
            raise SystemExit("logo.svg sem elemento id='fundo'")
        raiz.remove(fundo)
    return Rsvg.Handle.new_from_data(ET.tostring(raiz, encoding="utf-8"))


def desenhar(ctx: cairo.Context, svg: Rsvg.Handle, x: float, y: float, lado: float) -> None:
    """Desenha a tela de 108 unidades no quadrado (x, y, lado)."""
    ret = Rsvg.Rectangle()
    ret.x, ret.y, ret.width, ret.height = x, y, lado, lado
    svg.render_document(ctx, ret)


def superficie(px: int) -> tuple[cairo.ImageSurface, cairo.Context]:
    sup = cairo.ImageSurface(cairo.FORMAT_ARGB32, px, px)
    return sup, cairo.Context(sup)


def cor(hexa: str) -> tuple[float, float, float]:
    return tuple(int(hexa[i : i + 2], 16) / 255 for i in (1, 3, 5))


def forma(ctx: cairo.Context, px: int, redonda: bool) -> None:
    if redonda:
        ctx.arc(px / 2, px / 2, px / 2, 0, 2 * math.pi)
        return
    r = px * 0.22
    ctx.new_sub_path()
    ctx.arc(px - r, r, r, -math.pi / 2, 0)
    ctx.arc(px - r, px - r, r, 0, math.pi / 2)
    ctx.arc(r, px - r, r, math.pi / 2, math.pi)
    ctx.arc(r, r, r, math.pi, 3 * math.pi / 2)
    ctx.close_path()


def frente(svg: Rsvg.Handle, px: int) -> cairo.ImageSurface:
    sup, ctx = superficie(px)
    desenhar(ctx, svg, 0, 0, px)
    return sup


def composto(svg: Rsvg.Handle, px: int, redonda: bool) -> cairo.ImageSurface:
    """Simula o launcher: fundo + frente, parte visível (72/108) recortada pela máscara."""
    sup, ctx = superficie(px)
    forma(ctx, px, redonda)
    ctx.clip()
    ctx.set_source_rgb(*cor(COR_FUNDO))
    ctx.paint()
    escala = px * LADO / VISIVEL
    desenhar(ctx, svg, (px - escala) / 2, (px - escala) / 2, escala)
    return sup


def marca(svg: Rsvg.Handle, px: int) -> cairo.ImageSurface:
    """Só a marca, recortada no quadrado da zona segura (66/108), fundo transparente."""
    sup, ctx = superficie(px)
    escala = px * LADO / (2 * RAIO_SEGURO)
    desenhar(ctx, svg, (px - escala) / 2, (px - escala) / 2, escala)
    return sup


def conferir_zona_segura(svg: Rsvg.Handle) -> None:
    px = 432
    sup = frente(svg, px)
    passo = sup.get_stride()
    dados = sup.get_data()
    limite = RAIO_SEGURO * px / LADO
    centro = px / 2
    pior = 0.0
    for y in range(px):
        linha = y * passo
        for x in range(px):
            if dados[linha + 4 * x + 3]:  # alfa (ARGB32 little-endian: B G R A)
                pior = max(pior, math.hypot(x + 0.5 - centro, y + 0.5 - centro))
    if pior > limite + 1:
        raise SystemExit(f"marca fora da zona segura: {pior:.1f} px > {limite:.1f} px")
    print(f"zona segura ok: raio da marca {pior * LADO / px:.1f} dp de {RAIO_SEGURO:.0f} dp")


def gravar(sup: cairo.ImageSurface, caminho: Path) -> None:
    caminho.parent.mkdir(parents=True, exist_ok=True)
    sup.write_to_png(str(caminho))
    print(caminho.relative_to(APLICATIVO))


def gravar_texto(caminho: Path, texto: str) -> None:
    caminho.parent.mkdir(parents=True, exist_ok=True)
    caminho.write_text(texto, encoding="utf-8")
    print(caminho.relative_to(APLICATIVO))


ADAPTATIVO = """<?xml version="1.0" encoding="utf-8"?>
<!-- Gerado por ferramentas/gerar_icones.py a partir de assets/logo/logo.svg. -->
<adaptive-icon xmlns:android="http://schemas.android.com/apk/res/android">
    <background android:drawable="@color/ic_launcher_background"/>
    <foreground android:drawable="@mipmap/ic_launcher_foreground"/>
</adaptive-icon>
"""

COR_XML = f"""<?xml version="1.0" encoding="utf-8"?>
<!-- Gerado por ferramentas/gerar_icones.py: fundo do ícone adaptativo (fundo das telas). -->
<resources>
    <color name="ic_launcher_background">{COR_FUNDO}</color>
</resources>
"""


def gerar() -> None:
    sem_fundo = carregar(sem_fundo=True)
    conferir_zona_segura(sem_fundo)

    for nome, fator in DENSIDADES.items():
        pasta = RES / f"mipmap-{nome}"
        gravar(frente(sem_fundo, round(LADO * fator)), pasta / "ic_launcher_foreground.png")
        gravar(composto(sem_fundo, round(48 * fator), redonda=False), pasta / "ic_launcher.png")
        gravar(composto(sem_fundo, round(48 * fator), redonda=True), pasta / "ic_launcher_round.png")

    gravar_texto(RES / "mipmap-anydpi-v26" / "ic_launcher.xml", ADAPTATIVO)
    gravar_texto(RES / "mipmap-anydpi-v26" / "ic_launcher_round.xml", ADAPTATIVO)
    gravar_texto(RES / "values" / "ic_launcher_background.xml", COR_XML)

    pasta_logo = LOGO.parent
    gravar(marca(sem_fundo, LOGIN_DP), pasta_logo / "logo.png")
    gravar(marca(sem_fundo, LOGIN_DP * 2), pasta_logo / "2.0x" / "logo.png")
    gravar(marca(sem_fundo, LOGIN_DP * 3), pasta_logo / "3.0x" / "logo.png")


def para_pil(sup: cairo.ImageSurface):
    from PIL import Image

    sup.flush()
    tamanho = (sup.get_width(), sup.get_height())
    # ARGB32 do cairo: little-endian B G R A, alfa pré-multiplicado ("BGRa")
    dados = bytes(sup.get_data())
    return Image.frombuffer("RGBA", tamanho, dados, "raw", "BGRa", sup.get_stride(), 1).copy()


def ampliar(sup: cairo.ImageSurface, fator: int) -> cairo.ImageSurface:
    """Amplia sem interpolação, para enxergar os pixels do ícone de 48 px."""
    nova, ctx = superficie(sup.get_width() * fator)
    ctx.scale(fator, fator)
    ctx.set_source_surface(sup)
    ctx.get_source().set_filter(cairo.FILTER_NEAREST)
    ctx.paint()
    return nova


def previa(destino: Path) -> None:
    """Logo grande e os ícones redondo e quadrado em 48 px (também ampliado 4x) e 192 px,
    sobre fundo escuro e sobre fundo claro."""
    from PIL import Image, ImageDraw

    com_fundo = carregar(sem_fundo=False)
    sem_fundo = carregar(sem_fundo=True)
    img = Image.new("RGB", (1420, 620), "#3c4043")
    img.paste("#e8eaed", (1110, 0, 1420, 620))
    texto = ImageDraw.Draw(img)

    def colar(sup_ou_img, x: int, y: int) -> None:
        peca = sup_ou_img if isinstance(sup_ou_img, Image.Image) else para_pil(sup_ou_img)
        img.paste(peca, (x, y), peca)

    texto.text((40, 24), "logo.svg (512 px)", fill="white")
    colar(frente(com_fundo, 512), 40, 60)

    for i, (rotulo, redonda) in enumerate((("redondo", True), ("quadrado", False))):
        y = 60 + i * 280
        texto.text((600, y - 30), f"{rotulo}: 48 px | 48 px ampliado 4x | 192 px", fill="white")
        sup_pequeno = composto(sem_fundo, 48, redonda)
        pequeno = para_pil(sup_pequeno)
        colar(pequeno, 600, y)
        colar(ampliar(sup_pequeno, 4), 670, y)
        colar(composto(sem_fundo, 192, redonda), 890, y)
        texto.text((1140, y - 30), "sobre fundo claro", fill="black")
        colar(pequeno, 1140, y)
        colar(composto(sem_fundo, 192, redonda), 1210, y)

    destino.parent.mkdir(parents=True, exist_ok=True)
    img.save(destino)
    print(destino)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--previa", type=Path, help="grava também uma prévia em PNG")
    args = parser.parse_args()
    gerar()
    if args.previa:
        previa(args.previa)


if __name__ == "__main__":
    main()
