# Ověření 1. 10. 2026

## Kontrola před publikací zdrojů 0.7.0 — 4. 10. 2026

- Prošly `go test ./...` a `go vet ./...` všech šesti balíčků, včetně hystereze čerpadla, servisního času, restartu, změn režimu, chyb příkazů a opožděného potvrzení.
- Prošlo všech 11 frontendových testů, `vue-tsc --noEmit` a produkční Vite build. Prošel také integrační test vydávacího skriptu a actionlint.
- Nový produkční frontend ověřen v Chromiu na desktopu 1280 px a mobilu 390/340 px: bez horizontálního přetékání a chyb JavaScriptu. Vizuálně prohlédnut automatický i ruční režim. API fixtures ověřily změnu režimu, ruční Zap/Vyp, blokované Zap při neplatných datech a nepředaný výstup. Všechny povely obsluhovaly pouze anonymizované místní fixtures.
- Soukromý vstupní návod Kicony s interní adresou zůstává ignorovaný. Verze sjednocena na 0.7.0; publikace zdrojů do main neprovádí aktualizaci stable ani aktivaci v HA. Na živé instalaci nebyly odeslány příkazy ani upraven Node-RED.
- Aktuální oficiální dokumentace konfigurace HA apps, Ingress, Supervisor endpoints a MQTT switch znovu ověřena. Skutečná odezva Y04 a převzetí výlučného správce zůstávají součástí samostatného nasazení podle [postupu čerpadla](PUMP_CONTROL.md).

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

## Oprava čerstvosti HA — 3. 10. 2026 (místní, nevydaná)

- Uživatelské porovnání: stabilních 16 A má novější živý last_reported než čas zobrazený doplňkem. Aktuální oficiální zdroj HA 2026.9.4 ukazuje, že GET /states používá cached as_dict_json; při nezměněném stavu se aktualizuje objekt last_reported, nikoli tento JSON. Přesná verze Core uživatele zatím není doložená.
- Dosavadní GET snímek nahrazen jediným POST /core/api/template každých 5 s pro TUV i Teslu. Pevná šablona čte vybrané stavy a jejich časové vlastnosti přímo, bez as_dict, služeb, zápisu stavů nebo příkazů auta. Entity se předávají jako proměnné. Oprávnění homeassistant_api se nerozšiřuje. Starší odstavce o pouze GET popisují vydané verze 0.3.0/0.4.0.
- Regresní Go test: nezměněných 16 A a staré last_updated s čerstvým last_reported zůstává platné; skutečně starý report se odmítá i při novém HTTP příjmu; odmítnutí template endpointu vymaže snímek. Společný polling test kontroluje pevnou šablonu, vybrané entity a jeden požadavek.
- Go test ./... a go vet ./... prošly (nová dočasná build cache kvůli ACL původní cache). Kontrola publikovaných souborů a git diff --check prošly. Šablona ověřena lokálním Jinja renderem pro neměnnou hodnotu s novým reportem, chybějící entitu, prázdný výběr a filtrování atributů. Lokální Jinja kontrola nenahrazuje živé ověření v HA. Frontend ani veřejné API kontrakty se nemění. Bez zveřejnění a bez nasazení.
- Před implementací znovu ověřeny oficiální dokumentace apps configuration/communication/presentation, Supervisor endpoints, REST template a MQTT, plus HA core State/API (2026.9.4).

- Doplněna regrese TUV po uživatelském hlášení 3. 10.: neměnná spodní teplota 35,3 °C se starým last_updated a novým last_reported zůstává platná; skutečně staré hlášení se odmítá. Go test ./internal/homeassistant prošel. Společná oprava tedy zahrnuje čerstvost teplot TUV i Tesly.

## Hlavní živý přehled — 0.5.0 připravena 3. 10. 2026

- Hlavní Vue karty čtou výhradně /api/victron, /api/tuv a /api/tesla. Diagnostické karty sdílejí stejné snímky a neprovádějí další požadavky. API simulace se načítá až při rozbalení samostatné sekce. Rozhodovací Go jádro stále používá modelové vstupy; žádné aktivní výstupy ani živá historie se nepřipojují.
- Go test ./... a go vet ./... prošly. TypeScript/Vue kontrola a Vite produkční build prošly s existujícími lokálně připravenými dependencies odpovídajícími lockfile. Kvůli ACL node_modules byl build proveden nad kopií stejných zdrojů v artifacts/live-frontend-check s Vite configLoader runner. Žádné dependencies ani veřejné lockfile nebyly změněny.
- Playwright kontrola sestavené aplikace s anonymizovanými API fixtures: desktop 1280 px, mobil 390 px, úzký mobil 340 px, bez horizontálního přetečení a chyb konzole. Vizuálně prohlédnuty screenshoty desktop/mobil i zastaralý/rozporný stav.
- Významné regresní případy: platná nula výroby a proudu zůstává nulou; kladný/záporný tok baterie; skutečný a nastavený proud zůstávají oddělené; TUV 0 kW/2 kW aktivuje právě jeden celkový stupeň; rozporné přepínače neaktivují žádný. Jeden zastaralý string zneplatní celý součet výroby; zastaralá teplota a proud se skryjí; výpadek TUV zachová platnou Teslu; vypnutý MQTT nedostane modelový fallback. Výchozí přehled nevolá /api/state ani odesílací POST. Oddělený simulátor a diagnostika se dají rozbalit.
- Publication check prošel pro všech 59 zamýšlených veřejných souborů (včetně tří nových Vue/TS souborů), git diff --check prošel. Screenshoty a pomocný QA skript zůstávají v ignorovaných artifacts. Není vydáno ani nasazeno; interní HA template endpoint vyžaduje živé ověření po samostatném vydání.

## Kontrast stavů TUV — 0.5.1, 3. 10. 2026

- Vybraný celkový stupeň má kontrastní tmavý podklad a fajfku, ohřev nad 0 kW zelený, 0 kW tmavě šedý. Samostatný stavový pruh ukazuje vypnutý/zvolený/neznámý ohřev; neznámé stupně nejsou označené jako vypnuté. Čerpadlo i vypnuté řízení doplňkem mají textový a barevný štítek. Stále pouze hlášení HA.
- vue-tsc a produkční Vite build prošly; Playwright a vizuální kontrola na desktopu, mobilu 390/340 px a pro stavy 0/2 kW, zastaralou teplotu a konflikt prošly. Backend ani ovládání se nemění.

## Přesun technických popisků do diagnostiky — 0.5.2, 3. 10. 2026

- Z hlavních karet odstraněny časové značky, zdrojová hlášení, stavy transportu, detailní důvody kvality a vysvětlující poznámky. Názvy hodnot, jednotky, zvolený ohřev, stav čerpadla/nabíjení a kabelu zůstávají. Verze a vypnuté řízení přeneseny do rozbalovací sekce Diagnostika. Filtrace neplatných/zastaralých dat se nemění; pomlčka není nahrazena nulou.
- Produkční build/vue-tsc a vizuální kontrola desktop/mobil prošly. Playwright ověřil absenci technických popisků v hlavních kartách a zachování hlášení HA/limitů stáří v diagnostice, včetně regrese nul, zastaralosti, konfliktů, výpadků a simulace.

## 3. 10. 2026 – přehled 0.6.0

- Go testy a vet prošly. Nový čtecí výkon měničů: W/kW, skutečná nula, chybějící/NaN/záporná hodnota/jiná jednotka, hlášení v budoucnu nebo starší než 60 s, transport po 20 s a odmítnutí přístupu. Společný HA template požadavek vybírá TUV, Teslu a výkon měničů v jediném cyklu; API je chráněné Ingressem a nevystavuje tokeny.
- Vue-tsc a produkční Vite build prošly. Playwright s anonymizovanými fixtures ověřil desktop 1280 px, mobil 390 px a úzký mobil 340 px bez přetečení, pevná maxima 8/7/7 kW, SOC, nabíjení/vybíjení, nuly, nedostupná data, TUV konflikt i nezávislý výpadek. Překročení maxima sytí bargraf a zachovává plnou hodnotu. Žádné chyby konzole ani požadavky na ovládání nebo modelové API. Snímky vizuálně zkontrolované.
- Úvodní text, duplicitní karta domácí baterie a simulátor odstraněné z rozhraní; technické podklady zůstávají v Diagnostice. Živá entita sensor.vystupni_vykon vyžaduje uživatelské porovnání po aktualizaci; lokální kontrola používá fixtures.

## 3. 10. 2026 – MQTT výstup měničů 0.6.1

- Uživatel potvrdil topic system/0/Ac/ConsumptionOnOutput/L1/Power. Přidán do existujícího čtecího MQTT adaptéru jako inverter v W. Odstraněno čtení tohoto výkonu z HA, samostatný API požadavek i samostatná diagnostická karta; měření sdílí Victron.
- Go testy a vet prošly. Nové regrese: doplnění topicu do starší konfigurace bez přepsání uložených měření, zachování vlastního nebo prázdného topicu, nula, desetinný výkon, výkon nad limitem bargrafu, záporný/null/nečíselný/nekonečný vstup, stáří, retained, výpadek a čekání po resetu.
- Vue-tsc a Vite build prošly. Vizuální desktop/mobil/340 px kontrola s fixtures a Playwright ověřila stejný vzhled, příjem z Victronu, nulu, zastaralý výkon, výpadky a absenci požadavku na /api/power (HA fallback). Limity 8/7/7 kW a plná číselná hodnota nad rozsahem zachovány. Živý příjem nového topicu v doplňku musí potvrdit uživatel po aktualizaci.

## 3. 10. 2026 – lokální odhad nabití TUV (nevydáno)

- Převzat finální opravený model ze sdíleného vlákna: 170 vrstev se vzorkem v půlce vrstvy, výška 170 cm, čidla 40/130 cm, hranice 45–57 °C, horní minimum, omezení každé vrstvy a zaokrouhlení na 0,1 %. Používá stávající živé teploty dashboardu.
- Čtyři skupiny testů v web/tests/tuvCharge.test.ts prošly: opravené příklady (60/45 → 55,3 %, 60/50 → 75,5 %), meze 0/100 %, horní minimum včetně převrácení teplot, neplatné vstupy a monotónnost obou čidel. Spuštění node --test nebo pnpm test; vue-tsc a Vite build prošly.
- Lokální browser kontrola s anonymizovanými fixtures ověřila polohu procent uprostřed mezi čidly na desktopu i mobilu 390/340 px bez přetečení, opravený příklad, nulu/plné nabití a pomlčku při stale/invalid/offline. Diagnostika vysvětluje model a omezení; konzole bez chyb. Žádné změny živého HA nebo Node-RED.
- Výslovný pokyn uživatele: zatím nepublikovat, další GitHub zveřejnění až po jeho novém pokynu. Verze ponechána 0.6.1, žádný commit/push/release.

## 3. 10. 2026 – lokální síťové fáze a rozměry (nevydáno)

- Potvrzené MQTT topics Ac/Grid/L1/Power a odpovídající L2/L3 přidány do existujícího čtecího adaptéru, options/schema a diagnostiky; pod měniči se zobrazují pouze čísla ve W, bez bargrafu. Znaménko zachované. Starší konfigurace doplňuje chybějící klíče; explicitní prázdná/vlastní nastavení nepřepisuje.
- Go test/vet prošly, včetně regrese migrace, nuly, přesnosti, záporného výkonu, invalid jedné fáze bez dopadu na další, stale, retained a výpadku spojení. Prošel i test předchozího nevydaného modelu TUV, vue-tsc a Vite build.
- Browser kontrola s fixtures: stejné rozměry tří horních karet (desktop přibližně 341×319 px; mobil 390 px 343×312 px; mobil 340 px 293×320 px), všechny 4 bargrafy 18 px, ikona měniče přítomná, fáze bez bargrafů. Zastará/chybějící/offline fáze se zobrazuje pomlčkou, žádné přetečení nebo chyby konzole. Živá data na instalaci nejsou tímto testem potvrzená.
- Úpravy zůstávají společně s odhadem nabití TUV pouze lokálně podle výslovného zákazu publikace. Verze stále 0.6.1, bez commitu/pushe/release.

## 4. 10. 2026 – lokální čtení Kicony TUV (nevydáno)

- Nový select výkonu místo odstraněných spínačů, přesné volby 0/1/2/3 kW bez fallbacku. Stav systému, X16, Y15 a uptime sdílejí stávající čtecí template cyklus s TUV/Teslou. Horní teplota z jiného zařízení zůstala sensor.tepla_voda podle výslovného pokynu uživatele; TUV2 se nečte.
- Go test ./... a go vet ./... prošly (lokální Go runtime, nová izolovaná build cache). Regrese ověřily validní nulu, neznámé/chybějící selecty, všechny dokumentované blokace, limit stáří periodických hlášení a budoucí čas, souběžné X16/Y15, výpadek transportu a migraci starých options se zachováním vlastních čidel/čerpadla.
- Osm frontendových testů prošlo, včetně modelu zásoby tepla a nových stavů. Vue-tsc --noEmit a produkční Vite build prošly v lokální kopii zdrojů se stávajícími závislostmi (configLoader runner kvůli ACL staré .vite-temp). Žádné instalace ani změny závislostí.
- Playwright s anonymizovanými fixtures a vizuální kontrola PNG: desktop 1280 px, mobil 390/340 px, všechny blokace, současné vstupy i zpoždění textového stavu, diagnostika, offline/stale a chybějící select. Bez přetečení, chyb konzole nebo akčních požadavků; frontend pouze GET tuv/tesla/victron. Obnova/porucha čidel skryje odhad nabití TUV, ostatní blokace ponechávají platné teplotní údaje. Screenshoty lokálně artifacts/kicony-*.png.
- Živý příjem nových entit ani aktuální firmware Y15 nejsou lokálními fixtures potvrzené. Žádné příkazy zařízení, heartbeat, odblokování, reset, změny Node-RED nebo nasazení. Verze ponechána 0.6.2, bez commitu/pushe/release.
- Před implementací ověřena aktuální oficiální dokumentace: [HA apps](https://developers.home-assistant.io/docs/apps/), [Ingress](https://developers.home-assistant.io/docs/apps/presentation/#ingress), [komunikace apps](https://developers.home-assistant.io/docs/apps/communication/), [Supervisor API](https://developers.home-assistant.io/docs/api/supervisor/endpoints/), [MQTT](https://www.home-assistant.io/integrations/mqtt/), [Select](https://www.home-assistant.io/integrations/select/). Architektura ani oprávnění se nemění.


## 4. 10. 2026 – dokončené lokální řízení Y04

- Backend: samostatná sekundová smyčka čerpadla, hystereze >58/<57 °C se spodním >=57 pro vypnutí, servis v 18:30 Europe/Prague 30 s od HA hlášení Zap, OR s procesním požadavkem. Datum servisu a centrální režim jsou v SQLite; ruční povel ani potvrzení se neobnovují.
- API/UI: dva pevné POST endpointy mode/pump chráněné Ingressem a X-FVE-TUV, striktní JSON bez libovolné entity/služby. Dashboard nabízí Automatika TUV / Ruční ovládání a pouze ruční Zap/Vyp čerpadla; odlišuje požadavek, odeslání a HA stav. Aktivní ovládání ohřevu včetně transportu, konfiguračního klíče a endpointu odstraněno z rozpracovaného návrhu. Ohřev dále řídí Node-RED.
- V aktuálním projektu úspěšné Go test ./... a go vet ./...; frontend pnpm test (11 testů) a pnpm build (vue-tsc + Vite). Přidány regrese ručního režimu, neplatných/stale vstupů, restartu s ručním OFF, přechodu režimů během servisu, selhání persistence režimu, výchozího povolení a zachování explicitního false. API testy ověřují token, Ingress, neznámé položky, více JSON dokumentů, chybějící on, odmítnutí nebezpečného ON a neexistenci /api/tuv/power i obecného service API. Dosavadní testy pokrývají hranice hystereze, DST, sloučení požadavků, timeout/opakování a opožděné potvrzení.
- Vizuální kontrola skutečného finálního buildu s anonymními fixtures v headless Chrome/Playwright: desktop 1280, mobil 390 a úzký 340 px bez vodorovného přesahu a chyb konzole. Ověřeny oba režimy, ruční Zap i Vyp, pevné endpointy s tokenem, čekání na potvrzení bez vymyšleného vypnutí, chybové stavy, firmware blokace, vypnuté předání a diagnostika. Snímky uloženy v outputs tohoto chatu. Bez živého HA spojení.
- Výchozí pump_control_enabled=true je připravené pro funkční předání. Před aktivací musí uživatel vypnout všechny Node-RED větve zapisující na Y04; výslovně uložené false se nemění. Bez options.json lokální běh zůstává bez řízení.
- Omezení: lokálně nebyly ověřeny fyzické relé/průtok, skutečný Supervisor přístup, stavová zpětná vazba HA, typ instalace ani místní watchdog Y04. HA/optimistický stav není důkaz průtoku. Testy byly pouze lokální; nic nepublikováno ani nenasazeno, verze zůstává 0.6.2. Překrývající se návrhy druhého chatu byly před sjednocením zálohovány do work/before-integration aktuálního chatu.
- Oficiální dokumentace znovu ověřená: [komunikace apps](https://developers.home-assistant.io/docs/apps/communication/), [konfigurace](https://developers.home-assistant.io/docs/apps/configuration/), [Ingress](https://developers.home-assistant.io/docs/apps/presentation/), [Supervisor](https://developers.home-assistant.io/docs/api/supervisor/endpoints/), [MQTT switch](https://www.home-assistant.io/integrations/switch.mqtt/).
