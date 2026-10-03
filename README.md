# FVE Controller pro Home Assistant

<img src="fve_controller/icon.png" alt="Ikona FVE Controller" width="96" height="96">

Go backend, Vue/TypeScript přehled a SQLite. Existující aplikace zabalená jako doplněk Home Assistantu. **Čtení Victronu přes MQTT, TUV a Tesly přes HA a oddělená simulace: observe_only=true, control_enabled=false.** Neodesílá žádné příkazy zařízením.

## Instalace

Do obchodu doplňků/apps přidejte repozitář:

[Instalační repozitář FVE Controller — stable](https://github.com/ZabojnikM/ha-fve-controller#stable)

```text
https://github.com/ZabojnikM/ha-fve-controller#stable
```

Do pole v HA kopírujte pouze čistou URL z bloku výše. Nevkládejte Markdown se závorkami `[text](url)` ani adresu stránky větve s `/tree/stable`; správná URL končí `#stable`.

Větev `stable` obsahuje vydané verze. `main` je vývojová větev, nepřidávejte ji do HA. Vydání se nabídne až po nativním sestavení a spuštění obrazů obou architektur a po ověření veřejného přístupu ke všem vrstvám GHCR.

1. V systémových informacích HA ověřte typ instalace a architekturu. Apps podporuje Home Assistant OS; samotný Home Assistant Container nemá Supervisor/obchod apps.
2. Nastavení → Aplikace (ve starších verzích Doplňky) → Obchod → nabídka ⋮ → Repozitáře. Vložte výše uvedenou URL včetně `#stable`.
3. Obnovte seznam, otevřete **FVE Controller**, zvolte Instalovat a po dokončení Spustit.
4. Otevřete webové rozhraní. Musí být označeno SLEDOVACÍ REŽIM. Od verze 0.5.0 hlavní karty zobrazují živá data; oddělený simulátor je v rozbalovací sekci. Tokeny ani přihlašovací údaje pro GHCR nejsou potřeba.

Uživatel 2. 10. 2026 potvrdil úspěšnou instalaci a následnou aktualizaci z 0.1.0 na 0.1.1 v Home Assistantu na Raspberry Pi 4. Přesný typ instalace a architektura OS nejsou doložené ze systémových informací. Podrobnosti jsou v [záznamu ověření](docs/VALIDATION.md).

## Dokumentace

- [Podrobný návod a data](fve_controller/DOCS.md)
- [Vydání další verze a veřejné GHCR](docs/GITHUB.md)
- [Architektura](docs/ARCHITECTURE.md)
- [Implementační plán](docs/IMPLEMENTATION.md)
- [Ověření a jeho meze](docs/VALIDATION.md)
- [Changelog](fve_controller/CHANGELOG.md)

## Vývoj

Node 24.19.0, pnpm 11.25.0, Go 1.27.1. Z kořene:

```powershell
cd fve_controller/web
pnpm install --frozen-lockfile
pnpm build
cd ..
go test ./...
go run ./cmd/controller
```

Lokální UI: http://127.0.0.1:8099. Docker build context je přesně `fve_controller`:

```sh
docker build -t fve-controller:local fve_controller
python3 scripts/container_smoke.py fve-controller:local
```

Soukromé exporty, místní zadání, tokeny, databáze, logy a závislosti nejsou publikované. Povolené soubory Docker build contextu jsou vyjmenované v `.dockerignore`. Aktuální verze fyzické řízení neimplementuje.
