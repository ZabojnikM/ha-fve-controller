# Ikona FVE Controller

Schválený návrh „Energetický okruh“, vytvořený vestavěným ImageGen.
Originál: `fve-controller-original.png`. Odvozené PNG zachovávají celý obrázek a jsou pouze zmenšené bikubickou interpolací.

| Soubor | Rozměry | Použití |
| --- | --- | --- |
| `fve_controller/icon.png` | 128 × 128 | Obchod Home Assistant a README |
| `fve_controller/logo.png` | 256 × 256 | Detail doplňku Home Assistant |
| `fve_controller/web/public/app-icon.png` | 192 × 192 | Hlavička a větší ikona prohlížeče |
| `fve_controller/web/public/favicon.png` | 32 × 32 | Záložka prohlížeče |
| `fve_controller/web/public/apple-touch-icon.png` | 180 × 180 | Mobilní zástupce |

Cesty v tabulce jsou vůči kořeni repozitáře. Frontend používá relativní URL kvůli Ingress. Ikona mobilního zástupce sama nezajišťuje instalovatelnou PWA ani přihlášení mimo HA.

Home Assistant načítá `icon.png` a `logo.png` vedle `config.yaml`; viz [oficiální dokumentace](https://developers.home-assistant.io/docs/apps/presentation/). `panel_icon` je název MDI ikony, proto zůstává `mdi:solar-power`.

Původní zadání: moderní ikona FVE Controlleru se slunečním obloukem, smyčkou toků energie a bleskem, na tmavém zaobleném podkladu, bez textu.
