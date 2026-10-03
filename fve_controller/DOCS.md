# Provoz FVE Controller

## Odkaz pro instalaci v Home Assistantu

[Repozitář FVE Controller — stable](https://github.com/ZabojnikM/ha-fve-controller#stable)

Do pole Repozitáře v obchodě HA zkopírujte pouze tento text:

```text
https://github.com/ZabojnikM/ha-fve-controller#stable
```

Nevkládejte Markdown `[text](url)` ani adresu obsahující `/tree/stable`. Pokud clone skončí chybou „repository not found“, odstraňte chybný záznam a vložte čistou URL výše.

Uživatel 2. 10. 2026 potvrdil úspěšnou instalaci a spuštění. Jde o potvrzení běhu doplňku, nikoli o ověření fyzického řízení nebo všech funkcí Ingressu.

## Provoz

### Hlavní přehled (od verze 0.5.0)

Hlavní karty zobrazují skutečné údaje připojených zdrojů: výrobu tří stringů a domácí baterii z MQTT, obě teploty a jmenovitý stupeň TUV, čerpadlo a nabíjení Tesly z HA. Součet výroby se zobrazuje jen při platných údajích všech tří stringů. Platná nula zůstává nulou; chybějící, neplatné, zastaralé nebo přerušené čtení zobrazí pomlčku; důvod je v diagnostice. Výpadek jednoho zdroje neskrývá ostatní. Uživatel 2. 10. potvrdil kladný tok baterie při nabíjení a záporný při vybíjení.

Sekce **Diagnostika** obsahuje podrobné hodnoty, časové značky a limity stáří včetně článků baterie. Od verze 0.6.0 se simulátor v rozhraní nezobrazuje. Hlavní přehled nikdy nepřebírá chybějící hodnotu ze simulace. Výroba má bargraf do 8 kW, tok baterie a výkon měničů do 7 kW. Překročení zaplní bargraf, ale nesnižuje zobrazenou číselnou hodnotu. Baterie má také procentní bargraf SOC.

Doplněk stále pracuje pouze ve sledovacím režimu. Hlášený stupeň TUV není měřený příkon ani potvrzení fyzického sepnutí. Nastavený proud Tesly není skutečný odběr. Vzhled stupňů TUV je indikátor stavu, nikoli ovládací prvek. Předání fyzického řízení TUV vyžaduje samostatný výslovně povolený krok a ověření ochran i jediného vlastníka výstupů.

### Tesla přes Home Assistant (od verze 0.4.0)

Karta „Tesla · skutečná data“ čte existující entity HA. Výchozí mapování z původních podkladů je nastavitelné v `tesla_entities`: kabel `binary_sensor.nabijeci_kabel`, SOC `sensor.uroven_baterie`, stav nabíjení `sensor.nabijeni`, skutečný proud `sensor.proud_nabijecky`, výkon `sensor.vykon_nabijecky` a nastavený proud `number.nabijeci_proud`. Živou platnost těchto názvů ověřte po aktualizaci. Neexistující entita zneplatní pouze vlastní údaj. Neznámý údaj lze nechat prázdný; zobrazí se jako nenastavený.

`tesla_enabled` je výchozím nastavením true; `ha_enabled` povoluje společný čtecí adaptér pro TUV i Teslu. Obě karty používají jediný snímek HA každých 5 s a existující interní přístup Supervisoru. Nezadává se Tessie token, auto se neprobouzí a nevolají se žádné fyzické příkazy. Nastavený proud je pouze nastavení hlášené HA; není to skutečný proud, odeslaný příkaz našeho doplňku ani jeho potvrzení. Výkon se čte výhradně ze senzoru nabíječky, nikoli z elektroměru celé fáze nebo odhadu proud × napětí. Jednotky W a kW se převádějí na W; proud vyžaduje A a SOC %. Platná nula zůstává nulou.

Stav nabíjení podporuje standardní Tessie hodnoty `starting`, `charging`, `stopped`, `complete`, `disconnected`, `no_power` a původní `NoPower`; rozlišuje je od unknown/unavailable a neznámých hodnot. Přeložený text v UI HA není zdrojový stav entity. Při neplatném stavu se zobrazí pomlčka, nic se neodvozuje ze zapnutého přepínače nabíjení.

Každý údaj Tesly včetně kabelu má limit `tesla_fresh_seconds` (výchozí 900 s) podle `last_reported`, případně `last_updated`. Staré nebo budoucí časové razítko, unknown/unavailable, chybná jednotka a výpadek komunikace nejsou aktuální údaje. Příjem snímku HA sám nepotvrzuje čerstvost Tessie: integrace může znovu hlásit mezipaměť. U spícího auta může být karta zastaralá; doplněk ho kvůli čtení neprobouzí. Živé údaje nevstupují do simulace ani SQLite historie a API `/api/tesla` nezveřejňuje polohu, identifikátory auta, raw HA data nebo token.

Oficiální zdroje ověřené před implementací: [komunikace HA apps](https://developers.home-assistant.io/docs/apps/communication/), [REST API](https://developers.home-assistant.io/docs/api/rest/), [Tessie](https://www.home-assistant.io/integrations/tessie/), [stavy v integraci Tessie](https://github.com/home-assistant/core/blob/dev/homeassistant/components/tessie/const.py).

### TUV přes Home Assistant (od verze 0.3.0)

Karta „Teplá voda · skutečná data“ čte horní teplotu `sensor.tepla_voda`, spodní teplotu `sensor.tuv_1`, čerpadlo `switch.kicony_kc868_a16_y04` a celkové režimy `switch.tuv_0kw` až `switch.tuv_3kw`. Mapování je nastavitelné v `ha_entities`; `ha_enabled` má výchozí hodnotu true. Po aktualizaci restartujte doplněk. Není potřeba další heslo: backend použije `SUPERVISOR_TOKEN` a interní HA proxy. Konfigurace potřebuje `homeassistant_api: true`; Supervisor API oprávnění `hassio_api` zůstává vypnuté. Toto oprávnění HA samo o sobě není omezené na čtení; náš adaptér používá pouze POST `/core/api/template` s pevnou čtecí šablonou, bez volání služeb a změn stavů. HTTP POST zde pouze vyhodnocuje šablonu. Identifikátory vybraných entit se předávají jako proměnné, nikoli jako kód šablony; časy se čtou přímo z objektů stavů, aby se obešla serializační cache `/states`. TUV, Tesla a výkon měničů sdílejí jeden snímek každých 5 s. Pokud endpoint přístup odmítne, předchozí snímek se zneplatní; není fallback na uložený JSON `/states`.

Každých 5 s načte jeden snímek HA a ponechá pouze uvedené entity. Právě jeden zapnutý přepínač určuje jmenovitý výkon 0/1000/2000/3000 W. Žádný nebo více zapnutých přepínačů, chybějící entita a unknown/unavailable znamenají neurčený stav. TUV nemá měření příkonu: zvolený stupeň nepotvrzuje fyzické sepnutí spirál. Stav čerpadla je rovněž hlášení HA, nikoli měření průtoku.

Teploty musejí mít jednotku °C, číselnou hodnotu a čerstvý čas `last_reported` (u staršího HA fallback `last_updated`). Výchozí `ha_temperature_fresh_seconds` je 900 s. Čas hlášení HA nemusí být časem fyzického měření: integrační cache může stáří skrýt. U neměnných přepínačů se stáří změny nepoužívá jako timeout; ověřuje se aktuální příjem snímku. Po 20 s bez nového snímku nebo při chybě komunikace se hodnoty přestanou zobrazovat. Chybějící token a odmítnutí přístupu se zobrazí samostatně. Token, raw HA data a konfigurace se nevracejí přes API ani nelogují. Lokální běh bez tokenu zobrazuje chybějící přístup; token se nesmí ukládat do zdrojů.

Živé TUV je pouze informační karta na `GET /api/tuv`; nevstupuje do simulovaného jádra, balancování ani SQLite historie. Node-RED zůstává vlastníkem řízení a nemění se. Po aktualizaci porovnejte obě teploty, čerpadlo a zvolený stupeň s HA. Automatizované testy nenahrazují ověření interního Supervisor přístupu na cílové instalaci.

### Victron přes stávající MQTT most (od verze 0.2.0)

Samostatná karta „Victron · skutečná data“ čte z brokeru Mosquitto. Simulace pod ní zůstává oddělená: živé údaje nevstupují do doporučení, historie ani potvrzení balancování. Adaptér pouze odebírá přesné notifikační topics, nic nepublikuje. Stávající most musí nadále zajišťovat přísun dat a případný Victron keepalive; živý Node-RED se nemění.

V konfiguraci doplňku nastavte `mqtt_enabled: true`, broker `mqtt_host: core-mosquitto`, `mqtt_port: 1883` a účet pro MQTT. Heslo patří pouze do konfigurace HA (pole typu password); neposílejte ho do chatu ani do Git repozitáře. Připojení je určeno pro interní místní síť doplňků. TLS pro vzdálený broker v této etapě není implementováno.

Výchozí konfigurace již obsahuje potvrzené topics pro tuto instalaci. Maximální napětí může zůstat prázdné. Pro jinou instalaci topics změňte. Výchozí topics potvrzené pro tuto instalaci (zveřejněné se souhlasem vlastníka):

```yaml
mqtt_topics:
  min_cell: victron/N/c0619ab221ee/battery/99/System/MinCellVoltage
  max_cell: ""
  soc: victron/N/c0619ab221ee/battery/278/Soc
  battery: victron/N/c0619ab221ee/battery/278/Dc/0/Power
  solar_roof: victron/N/c0619ab221ee/solarcharger/0/Yield/Power
  solar_shelter: victron/N/c0619ab221ee/solarcharger/278/Yield/Power
  solar_fence: victron/N/c0619ab221ee/solarcharger/1/Yield/Power
```

Topics dalších měření ověřte v aktuálním brokeru; instance SmartShuntu, BMS a MPPT se mohou lišit. Očekávaná zpráva má tvar `{"value":3.274}`. Pro výkon baterie se zachovává původní znaménko; před využitím v regulaci musí být ověřeno na instalaci. Po úpravě konfigurace restartujte doplněk.

`mqtt_fresh_seconds` (výchozí 60) je limit od přijetí zprávy, nikoli důkaz stáří původního měření. Platná nula výkonu zůstává nulou. `null`, prázdná/poškozená zpráva a neplatný rozsah jsou neplatná data. Po výpadku nebo restartu se čeká na nové hodnoty; retained zpráva má neověřené stáří a nezobrazuje se jako aktuální měření. Most může přehrát staré údaje bez retained příznaku — před budoucím řízením ověřte jeho chování a dostupnost původního zdroje. Navýšení limitu stáří tuto nejistotu neřeší.

V lokálním vývoji lze cestu k backendovému `options.json` změnit přes `OPTIONS_PATH`; výchozí cesta je `DATA_DIR/options.json`. Soubor je ignorovaný v Gitu a konfigurace se nevrací přes API. Chybějící soubor znamená vypnuté MQTT. Živé vzorky jsou v paměti a na `GET /api/victron`; do SQLite se zatím neukládají. Maximální napětí článku je volitelné; při prázdném topicu se v živém panelu nezobrazuje. Při chybě přihlášení klient opakuje připojení; při chybě odběru zkontrolujte oprávnění účtu a restartujte doplněk.

Karta Solární výroba zobrazuje výkon stringů **Střecha, Přístřešek, Plot** a jejich součet ve W. Zatím jde o simulované hodnoty aktualizované každé 2 s, nikoli živé měření z HA/Victronu. Scénář Slunečný den ukazuje 2 600 + 1 400 + 600 = 4 600 W; Noc ukazuje platnou nulovou výrobu. Neplatný, chybějící nebo zastaralý vstup zneplatní součet (pomlčka); platné zbývající stringy zůstávají viditelné. Výpadek spojení skryje solární hodnoty. Výroba je informační údaj, nevstupuje do regulace a neukládá se do historie.

V lokálním vývoji: v `web` spusťte `pnpm install --frozen-lockfile` a `pnpm build`. Poté ze složky doplňku `go run ./cmd/controller`. Otevřete http://127.0.0.1:8099. Vyžaduje Go >=1.25, Node 24 a pnpm 11.25.0. SQLite se vytvoří v `data/simulation.db`. Proměnné DATA_DIR, WEB_DIR a LISTEN_ADDR mění cesty a adresu. Výchozí lokální server poslouchá pouze na loopbacku.

Pro Home Assistant OS nejprve ověřte typ instalace a CPU v systémových informacích. V Nastavení → Aplikace/Doplňky → Obchod → ⋮ → Repozitáře přidejte `https://github.com/ZabojnikM/ha-fve-controller#stable`. Obnovte obchod, vyberte FVE Controller, Instalovat, Spustit a Otevřít webové rozhraní. Větev stable vzniká pouze po úspěšném vydání. Home Assistant stáhne hotový veřejný obraz z GHCR bez přihlašování. Docker nastavuje INGRESS_ONLY=true a port 8099 je dostupný jen proxy 172.30.32.2; není publikován na hostiteli. Vydání zveřejní amd64 a aarch64 až po jejich nativním sestavení a testu. Skutečný Supervisor/Ingress je ještě nutné ověřit na cílovém systému. Home Assistant Container apps nepodporuje.

SQLite a současná nastavení (dokončení simulovaného balancování) jsou v `/data/simulation.db`; Supervisor zachovává `/data` při restartu i aktualizaci. Dočasné soubory SQLite WAL/SHM patří do stejného adresáře. Před aktualizací vytvořte zálohu v HA. Odinstalace není aktualizace a může data odstranit. API scénářů mění model jen do restartu, žádné nastavení fyzického řízení neexistuje.

Režimy observe_only=true a control_enabled=false jsou pevné vlastnosti této verze, nelze je přepnout konfigurací ani API. Implementovaný MQTT klient pouze čte telemetrii; fyzické příkazy neexistují. Scénáře mění jen modelová data. „Simulovaná skutečnost“ není měření fyzického zařízení. Požadavky se simulovanou skutečností nehýbou; tato verze není fyzikální model uzavřené regulační smyčky.

Historie se ukládá každé 2 s, uchovává 7 dní, API vrací posledních 120 vzorků. Balancování ukládá jen modelové dokončení pod samostatným klíčem. Bez předchozího dokončení je balancování splatné hned. Pro ukázku průzkumného kroku MPPT nejdřív dokončete simulované balancování (5 minut), pak vyberte Omezené MPPT. Ruční TUV zůstává ruční, dokud uživatel explicitně nezvolí jiný scénář; i ruční požadavek podléhá rozpočtu a ochranám.

Časování nevyřízených příkazů má izolovaný testovací model Tracker; není připojeno k výstupům ani API. Odeslaný výkon je vždy null. Načtení SQLite po restartu neobnovuje potvrzení příkazů. Před použitím skutečných vstupů v jádru doplnit zbývající čtecí adaptéry, ověřit znaménka, časová razítka a zpětnou vazbu. Aktivní řízení vyžaduje další implementaci, explicitní povolení a samostatné nasazení.

## Výkon měničů

Pevně vybraná entita `sensor.vystupni_vykon` se čte ve stejném cyklu HA každých 5 s a vystavuje na `GET /api/power`. Přijímají se nezáporná čísla v jednotkách W nebo kW (normalizace na W). Limit stáří hlášení je 60 s; příjem snímku má limit 20 s. Chybějící, neplatné nebo zastaralé hodnoty se zobrazují jako pomlčka. Toto čtení nemění stav zařízení ani konfiguraci HA. Oficiální podklady ověřeny 3. 10. 2026: [konfigurace apps](https://developers.home-assistant.io/docs/apps/configuration/), [Ingress](https://developers.home-assistant.io/docs/apps/presentation/), [Supervisor](https://developers.home-assistant.io/docs/api/supervisor/endpoints/), [MQTT](https://www.home-assistant.io/integrations/mqtt/).
