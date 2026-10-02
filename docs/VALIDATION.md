# Ověření 1. 10. 2026

- Go 1.27.1: `go test -cover ./...` úspěšný; jádro 96,9 %, HTTP/server 39,8 % statement coverage. `go vet ./...` bez nálezu.
- Sestaven Windows backend a Linux amd64 i arm64 s CGO_ENABLED=0. SQLite je pure Go.
- `pnpm build`: úspěšná kontrola TypeScriptu a produkční build Vite. Použité závislosti jsou v lockfile.
- Lokálně spuštěná skutečná Go aplikace s frontendem a SQLite; /health vrací ok, API potvrzuje observe_only=true a control_enabled=false.
- Testované hranice: platná nula versus neplatný/zastaralý vstup, znaménko baterie, vysoké SOC 95/93, kritické SOC 10/13, VT 16/20 a NT 16/17, společný limit a souběh, priorita TUV a uvolnění přídělu při teplotní blokaci, ochrany ručního požadavku, postupné přidávání a prodleva, pět minut souvislého balancování a reset při výpadku, interval 14 dní, restart, timeout a staré potvrzení i samostatná chyba příkazu.
- HTTP test: odmítnutí cizího peeru v Ingress včetně podvrženého X-Forwarded-For, API scénáře, zápis historie do SQLite, nepřítomnost řídicího endpointu.
- Vizuální kontrola ve skutečném prohlížeči: desktop 1280 px a mobil 390 px. Ověřeny scénáře zastaralých dat (BLOKACE, chybějící hodnota zobrazena pomlčkou) a kritického SOC (síť+nabíjení, nulové doporučení spotřebičům). Bez chyb a varování konzole při kontrole.

## Meze ověření
Docker není na tomto počítači dostupný: nebyl sestaven ani spuštěn finální kontejner. Linux binárky jsou cross-build, nebyly zde spuštěny. Skutečný HA Supervisor/Ingress nebyl dostupný a doplněk nebyl nasazen. Typ a architektura uživatelovy instalace zůstávají povinným ověřením před nasazením. Čtecí adaptéry ani aktivní řízení nejsou implementované.

Historie uchovává pouze kompletní platné čerstvé vzorky; výpadky nejsou ukládány jako měření. Jde o první model rozhodování, nikoli potvrzení fyzikální stability na skutečné instalaci. Limity a časování musejí projít měřením před případným řízením. Testovací Tracker není fyzický transport.

## Příprava publikování

- Zkontrolován explicitní seznam veřejných souborů; lokální soukromé zadání, exporty, databáze, nástroje a závislosti jsou mimo Git i Docker kontext. Před inicializací neexistovala Git historie ani remote.
- Po úpravě balení znovu prošel frontend build, Go testy a go vet.
- actionlint ověřil syntaxi a použití GitHub Actions bez nálezu.
- Integrační test vydávacího skriptu na dočasném bare Git repozitáři ověřil první vydání, aktualizaci se zachováním historie, opakování a odmítnutí zastaralého zdrojového SHA.
- Konečný obraz pro každou architekturu musí ještě projít nativními CI úlohami; teprve jejich úspěch dovolí vznik/aktualizaci stable.

## Vydání 0.1.0 — 2. 10. 2026

První vydání je zveřejněné: https://github.com/ZabojnikM/ha-fve-controller/releases/tag/v0.1.0 . Instalační URL: https://github.com/ZabojnikM/ha-fve-controller#stable .

- [Kontrola projektu](https://github.com/ZabojnikM/ha-fve-controller/actions/runs/36920203567): úspěch na nativních amd64 i aarch64 runnerech.
- [Vydávací workflow](https://github.com/ZabojnikM/ha-fve-controller/actions/runs/36966160957): úspěch všech úloh, včetně skutečného sestavení a spuštění obou kontejnerů, frontend buildu, Go testů/vet, SQLite při výměně kontejneru, Ingress ACL a relativních assetů.
- Anonymní stažení všech vrstev ověřeno v CI i nezávisle z vývojového počítače. Obraz `ghcr.io/zabojnikm/ha-fve-controller:0.1.0` je dostupný bez přihlášení.
- amd64 digest: `sha256:029904a98a5dbd79ff412b894e0c7c13d1dccc8efbff91056559db923f40e960`.
- aarch64 (OCI arm64) digest: `sha256:39635e4090eb030766f2ce6abaaecda78f17b262dac6652f929e06a8c081fa10`.
- Tag v0.1.0 a stable mají shodný Git strom. Stable vznikl až po úspěšném ověření GHCR. Nepoužit force push.
- Veřejný obsah prošel kontrolou sledovaných souborů. Soukromé podklady nejsou v žádném publikovaném commitu.

Toto doplňuje starší lokální výsledky výše: Docker je nyní ověřen v CI, nikoli na místním Windows. Nadále nebyla provedena instalace přes skutečný Home Assistant Supervisor ani ověřeno uživatelovo HA/CPU. Zůstává čistá simulace bez fyzického ovládání. CI upozorňuje na starší runtime použitých akcí checkout@v4/buildx@v3; běhy úspěšně proběhly na vynuceném Node 24.
