
  ALLO  1.0.1 ............................................ type specimen

  A friendly sans-serif typeface by Josh Puckett.
  Nine weights with obliques, and a variable font with three axes.


  AXES  (variable font)
    wght   100 ●━━━━━━━━━○━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━● 900
                         400
    opsz     9 ●━━━━━━━━━○━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━● 32
               small     14 text                     display
    ROND     0 ●━━━━━━━━━━━━━━━━━━━━━━━○━━━━━━━━━━━━━━━━━━━━● 100
               sharp               friendly               round

    Display (32) is drawn and spaced for headlines; below 14 only the
    spacing opens up. The static fonts are Text size, Friendly.


  STYLES
    100 Thin        400 Regular     700 Bold
    200 ExtraLight  500 Medium      800 ExtraBold
    300 Light       600 SemiBold    900 Black
                                    + an Oblique of each


  FEATURES
    ss01   Single-story a       a, and its accented forms
    ss02   Disambiguation       I l 1 0  →  serifed I, tailed l,
                                footed 1, slashed zero
    ss03   Open figures         4 6 9 with open forms
    cv01   Footed 1             cv02   Slashed zero
    cv03   Tailed l             cv04   Serifed I
    cv05   Dotted zero

    tnum   Tabular figures      every figure the same width
    zero   Slashed zero
    frac   Fractions            1/2 → ½   (1 1/2 and 11/14/2026 stay)
    sups   Superscript          2 → ²
    subs   Subscript            2 → ₂
    numr   Numerators           dnom   Denominators
    case   Capital forms        punctuation and signs sit with
                                capitals: (HELLO) [A–Z] ¿QUÉ?
    liga   Ligatures            tt ff tf ft share one crossbar
    calt   Contextual           automatic, on by default:
             typed arrows       a -> b   a <- b   a --> b
             lifting            (A) [1] −$4.50 +2% ¡HOLA! @ACME
    kern   Kerning

    Arrows need a space or nothing either side, so p->next stays code.
    Letter-spacing turns ligatures off in most browsers.


  CHARACTERS  (576)
    Latin        249 letters: A–Z a–z, accented forms, ẞ ß Æ æ Œ œ
    Figures      0–9  ⁰¹²³⁴⁵⁶⁷⁸⁹  ₀₁₂₃₄₅₆₇₈₉  ½ ⅓ ¼ ⅔ ¾ ⅛ ⅜ ⅝ ⅞
    Punctuation  ! ? ¡ ¿ … · • † ‡ § ¶ ‘ ’ “ ” « » ‹ › ‐ – — ‒
    Currency     $ ¢ £ ¥ € ₹ ₩ ₽ ₺ ₪ ฿ ₫ ₴ ₦ ₱ ₿ ƒ
    Math         + − × ÷ = ≠ ≈ < > ≤ ≥ ± ∞ ¬ ° ′ ″ ‰ № ℃ ℉
    Arrows       ← → ↑ ↓ ↔ ↕ ↖ ↗ ↘ ↙ ⇐ ⇒ ⇔ ↩ ↵ ↻ ↺
    Keys         ⌘ ⌥ ⇧ ⌃ ⎋ ⇥ ⌫ ⌦ ⏎ ␣ ⌕
    Shapes       ● ○ ■ □ ▲ ▼ ▶ ◀ ◆ ★ ♥ ⚠ ✓ ✕ ✗ ❮ ❯


  FILES
    fonts/otf/        18 static fonts. Install these on your computer.
    fonts/variable/   AlloVariable.ttf, for apps with variable fonts.
                      Its family is "Allo Variable", so it installs
                      beside the static fonts.
    fonts/webfonts/   The same fonts as WOFF2, for websites, plus
                      AlloVariableFriendly.woff2: the variable font
                      with roundness fixed at Friendly (50). About half
                      the size; it has no ROND axis.
    OFL.txt           The license.

    Mac       double-click a font, then Install
    Windows   select the files, right-click, Install


  WEB
    @font-face {
      font-family: "Allo";
      src: url("AlloVariable.woff2") format("woff2");
      font-weight: 100 900;
    }
    body { font-family: "Allo", sans-serif; }
    /* Optical size follows the font size on its own.
       To keep Text everywhere: font-optical-sizing: none; */
    .round { font-variation-settings: "ROND" 100; }

    Not changing roundness? Use AlloVariableFriendly.woff2 instead:
    about half the size, optical size and weight still work, and
    roundness stays at 50 ("ROND" does nothing).


  ....................................................................
  Josh Puckett  ·  joshpuckett.me  ·  github.com/joshpuckett/allo
  SIL Open Font License 1.1
