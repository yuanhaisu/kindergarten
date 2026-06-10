"""Generate the Nurture Planet logo - refined with Gaussian-blur gradients and crisp shapes."""
import math
from PIL import Image, ImageDraw, ImageFilter, ImageChops

# Supersampling: render at 3x then downscale
SCALE = 3
SIZE = 1024
S = SIZE * SCALE  # supersampled canvas size
C = S // 2         # center

# --- Tender Cosmos Palette ---
BLUE_SURFACE  = (91, 155, 213)   # warm cerulean
BLUE_SHADOW   = (46, 100, 160)   # deep shadow edge
BLUE_HIGHLIGHT = (190, 220, 242)  # soft highlight
GREEN_LEAF    = (76, 175, 80)     # tender sprout
GREEN_DARK    = (46, 125, 50)     # leaf depth
GREEN_BRIGHT  = (139, 210, 142)   # leaf highlight
ORANGE_MAIN   = (255, 152, 0)     # amber
ORANGE_GLOW   = (255, 200, 100)   # star halo
WHITE         = (255, 255, 255)

PLANET_R = int(S * 0.32)  # planet radius at supersampled size
PLANET_CY = C + S // 12   # planet center Y slightly below midpoint
PADDING = int(S * 0.10)   # safe padding from edges


def gaussian_gradient_sphere(size):
    """Create a sphere with smooth radial gradient using Gaussian blur trick.
    Returns an RGBA image with the sphere on transparent background."""
    pad = int(size * 0.3)
    img_size = size + pad * 2
    img = Image.new("RGBA", (img_size, img_size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    # Draw base solid circle
    draw.ellipse([pad, pad, pad + size, pad + size], fill=BLUE_SURFACE + (255,))

    # Create dark edge mask via Gaussian blur on a ring
    mask = Image.new("L", (img_size, img_size), 0)
    mask_draw = ImageDraw.Draw(mask)
    mask_draw.ellipse([pad, pad, pad + size, pad + size], fill=255)
    # Erode the mask to get just the edge
    inner_pad = pad + int(size * 0.08)
    mask_draw.ellipse([inner_pad, inner_pad, pad + size - int(size * 0.08),
                        pad + size - int(size * 0.08)], fill=0)
    mask = mask.filter(ImageFilter.GaussianBlur(radius=size * 0.12))

    # Apply shadow tint using the blurred edge mask
    shadow_layer = Image.new("RGBA", (img_size, img_size), BLUE_SHADOW + (0,))
    for y in range(img_size):
        for x in range(img_size):
            m = mask.getpixel((x, y))
            if m > 0:
                shadow_layer.putpixel((x, y), BLUE_SHADOW + (m,))
    img = Image.alpha_composite(img, shadow_layer)

    # Highlight spot (upper-left area)
    hl_mask = Image.new("L", (img_size, img_size), 0)
    hl_draw = ImageDraw.Draw(hl_mask)
    hl_cx = pad + size * 0.32
    hl_cy = pad + size * 0.28
    hl_r = int(size * 0.35)
    hl_draw.ellipse([int(hl_cx - hl_r), int(hl_cy - hl_r),
                      int(hl_cx + hl_r), int(hl_cy + hl_r)], fill=255)
    hl_mask = hl_mask.filter(ImageFilter.GaussianBlur(radius=size * 0.1))

    highlight_layer = Image.new("RGBA", (img_size, img_size), (0, 0, 0, 0))
    for y in range(img_size):
        for x in range(img_size):
            m = hl_mask.getpixel((x, y))
            if m > 0:
                highlight_layer.putpixel((x, y), WHITE + (int(m * 0.55),))
    img = Image.alpha_composite(img, highlight_layer)

    # Secondary small bright spot
    hl2_mask = Image.new("L", (img_size, img_size), 0)
    hl2_draw = ImageDraw.Draw(hl2_mask)
    hl2_cx = pad + size * 0.40
    hl2_cy = pad + size * 0.24
    hl2_r = int(size * 0.1)
    hl2_draw.ellipse([int(hl2_cx - hl2_r), int(hl2_cy - hl2_r),
                       int(hl2_cx + hl2_r), int(hl2_cy + hl2_r)], fill=255)
    hl2_mask = hl2_mask.filter(ImageFilter.GaussianBlur(radius=size * 0.04))
    for y in range(img_size):
        for x in range(img_size):
            m = hl2_mask.getpixel((x, y))
            if m > 0:
                r, g, b, a = img.getpixel((x, y))
                blend = int(m * 0.35)
                img.putpixel((x, y), (
                    min(255, r + blend),
                    min(255, g + blend),
                    min(255, b + blend),
                    a
                ))

    return img


def create_star_image(outer_r, inner_r, color, rotation=0):
    """Create a soft rounded star as a separate RGBA image."""
    size = outer_r * 3
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)
    cx = size // 2
    cy = size // 2
    points = 4

    vertices = []
    for i in range(points * 2):
        angle = rotation + (i * math.pi / points) - math.pi / 2
        r = outer_r if i % 2 == 0 else inner_r
        x = cx + r * math.cos(angle)
        y = cy + r * math.sin(angle)
        vertices.append((x, y))

    draw.polygon(vertices, fill=color)

    # Soften via blur
    img = img.filter(ImageFilter.GaussianBlur(radius=outer_r * 0.15))

    # Redraw sharper center
    img2 = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw2 = ImageDraw.Draw(img2)
    vertices2 = []
    for i in range(points * 2):
        angle = rotation + (i * math.pi / points) - math.pi / 2
        r = (outer_r * 0.7) if i % 2 == 0 else (inner_r * 1.2)
        x = cx + r * math.cos(angle)
        y = cy + r * math.sin(angle)
        vertices2.append((x, y))
    draw2.polygon(vertices2, fill=color)
    img2 = img2.filter(ImageFilter.GaussianBlur(radius=outer_r * 0.06))

    img = Image.alpha_composite(img, img2)

    return img


def create_sprout_image(scale_factor):
    """Create the sprout as a separate RGBA image that will be composited."""
    s = scale_factor
    # Canvas big enough for the sprout
    w = int(s * 3.2)
    h = int(s * 5.5)
    img = Image.new("RGBA", (w, h), (0, 0, 0, 0))

    # Helper to draw soft ellipse on the sprout canvas
    def soft_ellipse(im, x, y, rx, ry, color, blur_radius=None):
        mask = Image.new("L", im.size, 0)
        md = ImageDraw.Draw(mask)
        md.ellipse([int(x - rx), int(y - ry), int(x + rx), int(y + ry)], fill=255)
        if blur_radius:
            mask = mask.filter(ImageFilter.GaussianBlur(radius=blur_radius))
        colored = Image.new("RGBA", im.size, color + (0,))
        for py in range(im.size[1]):
            for px in range(im.size[0]):
                m = mask.getpixel((px, py))
                if m > 0:
                    colored.putpixel((px, py), color + (m,))
        return Image.alpha_composite(im, colored)

    # Origin point (where sprout meets planet) is at bottom center
    ox = w // 2
    oy = h - int(s * 0.1)

    # Stem - curved, growing up and slightly right
    stem_cx = ox + int(s * 0.15)
    stem_top = int(oy - s * 2.8)
    stem_bot = oy
    stem_w = int(s * 0.42)

    # Draw stem as a series of overlapping soft circles
    steps = 50
    for i in range(steps + 1):
        t = i / steps
        # Gentle S-curve
        bx = ox + int(s * 0.25 * math.sin(t * math.pi * 0.7))
        by = stem_bot - int((stem_bot - stem_top) * t)
        r = stem_w * (1 - t * 0.55)
        if r > 3:
            img = soft_ellipse(img, bx, by, r, r * 0.8, GREEN_LEAF, blur_radius=r * 0.15)

    # Stem highlight line
    for i in range(0, steps + 1, 2):
        t = i / steps
        bx = ox + int(s * 0.25 * math.sin(t * math.pi * 0.7))
        by = stem_bot - int((stem_bot - stem_top) * t)
        r = stem_w * (1 - t * 0.55) * 0.35
        if r > 1.5:
            img = soft_ellipse(img, bx - int(r * 0.5), by, r, r * 0.5,
                               GREEN_BRIGHT, blur_radius=r * 0.25)

    # Left leaf (larger, drooping slightly)
    leaf_lx = ox - int(s * 0.5)
    leaf_ly = stem_bot - int(s * 1.50)
    leaf_rx = int(s * 1.2)
    leaf_ry = int(s * 0.35)
    img = soft_ellipse(img, leaf_lx, leaf_ly, leaf_rx, leaf_ry, GREEN_DARK, blur_radius=s * 0.06)
    img = soft_ellipse(img, leaf_lx, leaf_ly, leaf_rx * 0.85, leaf_ry * 0.7,
                       GREEN_LEAF, blur_radius=s * 0.04)
    # Leaf vein
    img = soft_ellipse(img, leaf_lx - int(s * 0.15), leaf_ly,
                       int(s * 0.5), int(s * 0.08), GREEN_BRIGHT, blur_radius=s * 0.03)

    # Right leaf (smaller, reaching up)
    leaf2_x = ox + int(s * 0.55)
    leaf2_y = stem_bot - int(s * 2.0)
    # Tilted ellipse for the right leaf
    leaf2_rx = int(s * 0.9)
    leaf2_ry = int(s * 0.3)
    img = soft_ellipse(img, leaf2_x, leaf2_y, leaf2_rx, leaf2_ry, GREEN_DARK, blur_radius=s * 0.05)
    img = soft_ellipse(img, leaf2_x + int(s * 0.05), leaf2_y - int(s * 0.02),
                       int(leaf2_rx * 0.8), int(leaf2_ry * 0.65),
                       GREEN_LEAF, blur_radius=s * 0.03)
    img = soft_ellipse(img, leaf2_x + int(s * 0.1), leaf2_y,
                       int(s * 0.4), int(s * 0.06), GREEN_BRIGHT, blur_radius=s * 0.02)

    # Third small leaf/bud at the top
    leaf3_x = ox + int(s * 0.3)
    leaf3_y = stem_top + int(s * 0.15)
    img = soft_ellipse(img, leaf3_x, leaf3_y, int(s * 0.35), int(s * 0.18),
                       GREEN_LEAF, blur_radius=s * 0.04)
    img = soft_ellipse(img, leaf3_x, leaf3_y, int(s * 0.22), int(s * 0.1),
                       GREEN_BRIGHT, blur_radius=s * 0.02)

    return img


def draw_ring_directly(draw, cx, cy, ring_rx, ring_ry, ring_width, color):
    """Draw partial orbital ring arcs directly on the canvas behind the planet."""
    # Draw upper arc (behind planet, visible above)
    _draw_ellipse_arc(draw, cx, cy, ring_rx, ring_ry, ring_width, color, 170, 280)
    # Draw lower arc (behind planet, visible below)
    _draw_ellipse_arc(draw, cx, cy, ring_rx, ring_ry, ring_width, color, 260, 370)


def _draw_ellipse_arc(draw, cx, cy, rx, ry, width, color, start_deg, end_deg):
    """Draw a soft arc segment of an ellipse."""
    steps = 120
    for i in range(steps + 1):
        deg = start_deg + (end_deg - start_deg) * i / steps
        rad = math.radians(deg)
        x = cx + rx * math.cos(rad)
        y = cy + ry * math.sin(rad)
        # Vary alpha and size to create tapered ends
        edge_dist = min(i, steps - i)
        taper = min(1.0, edge_dist / (steps * 0.08))
        alpha = int(60 * taper)
        if alpha > 0:
            r = width + int(width * 0.5 * taper)
            draw.ellipse([int(x) - r, int(y) - r, int(x) + r, int(y) + r],
                        fill=color + (alpha,))


def create_logo():
    """Compose the full logo from layered elements."""
    canvas = Image.new("RGBA", (S, S), (0, 0, 0, 0))

    # --- 1. Planet Sphere ---
    sphere = gaussian_gradient_sphere(PLANET_R * 2 + int(PLANET_R * 0.1))
    sphere_place_x = C - sphere.width // 2
    sphere_place_y = PLANET_CY - sphere.height // 2
    canvas.paste(sphere, (sphere_place_x, sphere_place_y), sphere)

    # Planet actual bounds after placement
    px = sphere_place_x
    py = sphere_place_y
    pr = sphere.width // 2

    # --- 2. Sprout (growing from top of planet) ---
    sprout_scale = S / 480  # scale relative to canvas (smaller = less tall relative to planet)
    sprout = create_sprout_image(sprout_scale)
    # Position: base of sprout sits inside the top of the planet
    sprout_x = C - sprout.width // 2
    sprout_y = PLANET_CY - pr + int(pr * 0.02) - sprout.height + int(sprout_scale * 0.1)
    canvas.paste(sprout, (sprout_x, sprout_y), sprout)

    # --- 3. Stars ---
    star_defs = [
        # (x_rel, y_rel, outer_radius, rotation) - kept within padded area
        (C - int(S * 0.17), PLANET_CY - pr - int(S * 0.05), int(S * 0.042), 0.15),
        (C + int(S * 0.20), PLANET_CY - pr - int(S * 0.06), int(S * 0.048), 0.35),
        (C + int(S * 0.01), PLANET_CY - pr - int(S * 0.14), int(S * 0.032), -0.1),
        (C + int(S * 0.25), PLANET_CY + pr - int(S * 0.04), int(S * 0.028), 0.2),
        (C - int(S * 0.22), PLANET_CY - pr - int(S * 0.02), int(S * 0.026), 0.0),
    ]

    for sx, sy, outer_r, rot in star_defs:
        sx, sy = int(sx), int(sy)
        inner_r = int(outer_r * 0.38)
        star_img = create_star_image(outer_r, inner_r, ORANGE_MAIN, rot)

        # Glow behind star
        glow = Image.new("RGBA", (int(outer_r * 6), int(outer_r * 6)), (0, 0, 0, 0))
        glow_draw = ImageDraw.Draw(glow)
        gc = glow.width // 2
        glow_draw.ellipse([gc - int(outer_r * 2), gc - int(outer_r * 2),
                           gc + int(outer_r * 2), gc + int(outer_r * 2)], fill=ORANGE_GLOW + (25,))
        glow = glow.filter(ImageFilter.GaussianBlur(radius=outer_r * 0.7))
        canvas.paste(glow, (sx - glow.width // 2, sy - glow.height // 2), glow)

        canvas.paste(star_img, (sx - star_img.width // 2, sy - star_img.height // 2), star_img)

        # Bright center dot
        dot_r = max(1, inner_r // 2)
        dot_img = Image.new("RGBA", (dot_r * 5, dot_r * 5), (0, 0, 0, 0))
        dot_draw = ImageDraw.Draw(dot_img)
        dc = dot_img.width // 2
        dot_draw.ellipse([dc - dot_r, dc - dot_r, dc + dot_r, dc + dot_r],
                         fill=(255, 235, 160, 230))
        dot_img = dot_img.filter(ImageFilter.GaussianBlur(radius=dot_r * 0.3))
        canvas.paste(dot_img, (sx - dot_img.width // 2, sy - dot_img.height // 2), dot_img)

    # --- 4. Small sparkle dots ---
    sparkles = [
        (C - S * 0.28, PLANET_CY, int(S * 0.010)),
        (C + S * 0.29, PLANET_CY - S * 0.05, int(S * 0.012)),
        (C + S * 0.16, PLANET_CY + pr + S * 0.06, int(S * 0.009)),
        (C - S * 0.20, PLANET_CY - pr - S * 0.18, int(S * 0.008)),
        (C + S * 0.31, PLANET_CY + S * 0.04, int(S * 0.007)),
        (C - S * 0.10, PLANET_CY + pr + S * 0.10, int(S * 0.010)),
    ]

    for dx, dy, dr in sparkles:
        dx, dy = int(dx), int(dy)
        sp_img = Image.new("RGBA", (int(dr * 6), int(dr * 6)), (0, 0, 0, 0))
        sp_draw = ImageDraw.Draw(sp_img)
        sc = sp_img.width // 2
        sp_draw.ellipse([sc - int(dr), sc - int(dr), sc + int(dr), sc + int(dr)], fill=ORANGE_GLOW + (120,))
        sp_img = sp_img.filter(ImageFilter.GaussianBlur(radius=float(dr * 0.3)))
        canvas.paste(sp_img, (dx - sp_img.width // 2, dy - sp_img.height // 2), sp_img)
        # Brighter center
        sp_draw.ellipse([sc - dr // 2, sc - dr // 2, sc + dr // 2, sc + dr // 2],
                         fill=ORANGE_MAIN + (200,))
        canvas.paste(sp_img, (dx - sp_img.width // 2, dy - sp_img.height // 2), sp_img)

    # --- 5. Subtle orbital ring behind planet ---
    ring_rx = pr + int(S * 0.04)
    ring_ry = ring_rx // 3
    ring_w = max(3, int(S * 0.012))
    ring_canvas = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    ring_draw = ImageDraw.Draw(ring_canvas, "RGBA")
    draw_ring_directly(ring_draw, C, PLANET_CY, ring_rx, ring_ry, ring_w, ORANGE_MAIN)

    # Composite: ring behind everything else
    canvas_final = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    canvas_final.paste(ring_canvas, (0, 0), ring_canvas)
    canvas_final.paste(canvas, (0, 0), canvas)

    # --- Downscale to final size ---
    result = canvas_final.resize((SIZE, SIZE), Image.LANCZOS)

    # Subtle sharpen
    result = result.filter(ImageFilter.UnsharpMask(radius=0.6, percent=50, threshold=2))

    return result


if __name__ == "__main__":
    logo = create_logo()
    output_path = r"D:\yuanhaisu\personal_projects\kindergarten\assets\logo.png"
    logo.save(output_path, "PNG")
    print(f"Logo saved to {output_path} ({logo.size[0]}x{logo.size[1]})")

    for fs in [32, 64, 128, 256]:
        fav = logo.resize((fs, fs), Image.LANCZOS)
        fp = rf"D:\yuanhaisu\personal_projects\kindergarten\assets\logo-{fs}.png"
        fav.save(fp, "PNG")
        print(f"  {fs}x{fs} saved")
