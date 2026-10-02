# Spuštění simulátoru

## Odkaz pro instalaci v Home Assistantu

[Repozitář FVE Controller — stable](https://github.com/ZabojnikM/ha-fve-controller#stable)

Do pole Repozitáře v obchodě HA zkopírujte pouze tento text:

```text
https://github.com/ZabojnikM/ha-fve-controller#stable
```

Nevkládejte Markdown `[text](url)` ani adresu obsahující `/tree/stable`. Pokud clone skončí chybou „repository not found“, odstraňte chybný záznam a vložte čistou URL výše.

Uživatel 2. 10. 2026 potvrdil úspěšnou instalaci a spuštění. Jde o potvrzení běhu doplňku, nikoli o ověření fyzického řízení nebo všech funkcí Ingressu.

## Provoz

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
