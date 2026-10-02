# Ověření 1. 10. 2026

## Čtecí adaptér Victron MQTT — místní úprava 2. 10. 2026

- Go testy ověřují rozlišení nuly, chybějící hodnoty, null/prázdného/poškozeného payloadu, zastarání, retained zprávy, výpadku a resetu po reconnectu. Znaménko baterie se zachovává; potvrzení znaménka na instalaci je stále otevřené.
- Lokální anonymizovaný MQTT broker fixture ověřil skutečnou cestu CONNECT → SUBSCRIBE → příjem výkonu → DISCONNECT; klient neposlal PUBLISH. Odmítnutý nebo chybějící SUBACK není úspěšný odběr. API je chráněné stejným Ingress ACL a neobsahuje přístupové údaje.
- Kontrola TypeScriptu a produkční Vite build prošly přímým spuštěním instalovaných nástrojů (pnpm wrapper vyžadoval nevyžádanou reinstalaci). Vizuálně ověřen desktop a mobil 390 × 844 px: platná nula, napětí 3,274 V, nenastavené topics a následné zastarání hodnot s pomlčkou. Mobil bez vodorovného přetékání, konzole bez chyb a varování.
- Oficiální dokumentace znovu ověřena: [apps konfigurace](https://developers.home-assistant.io/docs/apps/configuration/), [Ingress](https://developers.home-assistant.io/docs/apps/presentation/), [Supervisor API](https://developers.home-assistant.io/docs/api/supervisor/endpoints/), [HA MQTT](https://www.home-assistant.io/integrations/mqtt/) a [Victron dbus-flashmq](https://github.com/victronenergy/dbus-flashmq). Použit Eclipse Paho 1.5.1.
- Živé údaje jsou oddělené od simulátoru, jeho historie a balancování. MQTT nic nepublikuje; keepalive zůstává na existujícím mostu. Zdrojové stáří nelze určit pouze z času příjmu do brokeru. HA/CPU, most, autentizace a skutečné topics musejí být ověřené při nasazení. Úprava nebyla vydaná ani nasazená, uživatelův Mosquitto zatím nebyl připojen.

## Frontend inspirovaný evcc — místní úprava 2. 10. 2026

- Kontrola TypeScriptu (`vue-tsc --noEmit`) a produkční sestavení (`vite build`) prošly přímým spuštěním instalovaných nástrojů přes Node. Příkaz pnpm zastavil automatický pokus o přeinstalaci závislostí kvůli chybějícímu TTY; závislosti ani lockfile nebyly změněny.
- Vizuálně ověřen produkční frontend nad místním simulátorem: desktop a mobil 390 × 844 px, bez vodorovného přetékání. Zkontrolovány karty spotřebičů, noc s platnou nulou, zastaralá měření s pomlčkou a blokací a kritické SOC s doporučením síť+nabíjení. Konzole bez varování a chyb.
- Náhled je uložen místně v ignorovaném `artifacts/evcc-frontend-desktop.jpg`. Backend, protokol API a fyzické řízení touto úpravou nejsou měněny. Úprava nebyla vydána ani nasazena do HA.

## Aktualizace 0.1.0 → 0.1.1 — potvrzeno 2. 10. 2026

- [Vydání 0.1.1](https://github.com/ZabojnikM/ha-fve-controller/releases/tag/v0.1.1) obsahuje novou ikonu aplikace.
- [Kontrola projektu](https://github.com/ZabojnikM/ha-fve-controller/actions/runs/36967924500) i [vydávací workflow](https://github.com/ZabojnikM/ha-fve-controller/actions/runs/36968056247) úspěšně dokončily všechny úlohy. Zdrojový commit: `2592260c2c04d47ed48258fcc07169fb621f13f0`.
- Na nativních amd64 a aarch64 runnerech prošly sestavení, Go testy/vet, spuštění kontejnerů, zachování SQLite při výměně kontejneru, Ingress ACL a poskytování PNG ikon. Před aktualizací stable prošlo anonymní stažení obrazů.
- Uživatel následně potvrdil: „aktualizace funguje správně“. Tím je potvrzený praktický průchod vydáním a aktualizací v jeho Home Assistantu na Raspberry Pi 4.
- Přesný typ instalace HA a architekturu OS uživatel nedoložil. Záloha a zachování historie na jeho zařízení nebyly samostatně potvrzené; test perzistence výše je výsledek CI.
- Aplikace zůstává simulátorem v observe_only, bez příkazů fyzickým zařízením. Starší omezení níže popisují stav v době dané etapy.

## Historické výsledky

Aktualizace 2. 10. 2026: uživatel potvrdil, že je doplněk nainstalovaný v Home Assistantu a běží. Instalace uspěla s čistou URL `https://github.com/ZabojnikM/ha-fve-controller#stable`. Předchozí chyba klonování vznikla vložením Markdown odkazu a cesty `/tree/stable`. Typ HA, CPU a podrobný test Ingressu tímto potvrzením doloženy nejsou. Níže jsou zachované historické výsledky jednotlivých etap.

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

## Čtení TUV 0.3.0 — 2. 10. 2026

- Go testy všech balíčků a go vet prošly. Nové testy ověřují stupně 0–3 kW, žádný/více zapnutých přepínačů, missing/unknown/unavailable, nulovou teplotu, jednotky, stáří a čas v budoucnosti, timeout příjmu, restart, chybu přihlášení a obnovu.
- HTTP test ověřuje pouze GET, filtrování nevybraných entit a absenci tokenu, konfigurace a raw HA dat v API. Integrační API test ověřuje Ingress ACL pro /api/tuv. Nativní kontejnerový smoke test nově kontroluje bezpečný stav TUV bez Supervisor tokenu.
- Frontend vue-tsc a produkční Vite build prošly se zamčenými závislostmi. Lokální build proběhl v izolované ignorované složce .tools/frontend-check kvůli nekompatibilnímu původnímu node_modules.
- Vizuální kontrola v Chromiu: desktop 1280 px a mobil 390 px, bez horizontálního přetečení. Ověřeny platné anonymizované vzorky, zastaralá teplota, konfliktní stupeň, chybějící Supervisor token a chyba backendu. Bez chyb JavaScriptu.
- Lokálně není přístup k uživatelovu HA. Interní proxy, jednotky a aktualizace teplot na konkrétní instalaci ověří uživatel po aktualizaci. Živá data zůstávají oddělená od simulace, neukládají se do SQLite; Node-RED ani fyzické výstupy se nemění.
- Oficiální dokumentace HA apps konfigurace, komunikace, Ingress, Supervisor endpoints, REST API a MQTT byla ověřena v tomto sezení před implementací. Pro přístup k Core se používá homeassistant_api, nikoli rozšířené hassio_api.

## Čtení Tesly 0.4.0 — 2. 10. 2026

- Rozšířen stávající čtecí HA adaptér: TUV a Tesla sdílejí jeden GET snímek každých 5 s. Přímé Tessie API, probouzení auta a fyzické příkazy nejsou implementované.
- Go testy všech balíčků a go vet prošly. Anonymizované Tesla fixtures ověřují převod kW/W, platnou nulu, SOC a jednotky, neplatné/unknown/unavailable stavy, stáří zdroje a budoucí čas, timeout, obnovu, oddělení skutečného a nastaveného proudu, volitelné entity a kompatibilitu starých options.json.
- Test společného polling cyklu ověřuje pouze jeden GET pro TUV i Teslu a nepřenášení polohy nebo nevybraných údajů. API test chrání /api/tesla Ingressem a ověřuje absenci tokenu a konfigurace. Kontejnerový smoke test doplněn o bezpečný stav Tesly bez Supervisor tokenu.
- Vue/TypeScript a Vite produkční build prošly. Chromium desktop 1280 px a mobil 390 px bez horizontálního přetečení a chyb JavaScriptu; vizuálně ověřeno nabíjení, odpojení s platnou nulou, částečná/neplatná/zastaralá data a nedostupný backend. Znovu prošla kontrola karty TUV.
- Výchozí Tesla entity pocházejí z původního Node-RED a dashboardu, nikoli z místního živého HA. Aktuální mapování a jednotky ověří uživatel na cílové instalaci. HA čas hlášení není důkazem čerstvosti fyzického měření Tessie.
- Ověřeny oficiální REST/komunikační dokumentace HA a zdrojový seznam charge states integrace Tessie. Živá doporučení a historie zůstávají nepřipojené.
