# Changelog

## 0.7.1

- Odstraněny i zbývající časové kontroly ostatních HA údajů: Tesla, stav firmware, uptime a stáří přijatého snímku. Chyby spojení a neplatné/nedostupné hodnoty se dál odmítají; čas hlášení je pouze informační.
- Victron MQTT zachovává dosavadní kontrolu stáří jednotlivých měření. Timeouty příkazů a potvrzení čerpadla zůstávají.

- Teploty TUV se neodmítají podle stáří hlášení HA; odstraněn limit 120 s pro čerpadlo i 900 s pro zobrazení a odhad zásoby tepla.
- Nedostupné/neplatné teploty, chybné jednotky, výpadek čtení HA a odpojené napájení čidel nadále blokují zapnutí čerpadla. Časy hlášení zůstávají informační.
- Odstraněny nastavení časových limitů teplot; staré uložené položky se ignorují.

## 0.7.0

- Funkční samostatné řízení čerpadla: hystereze 58/57 °C, servisní protočení 18:30 Europe/Prague na 30 s sloučené s procesním požadavkem.
- Centrální Automatika TUV / Ruční ovládání, Zap/Vyp čerpadla, ukládání režimu a oddělené potvrzení HA. Ruční Zap se po restartu neobnovuje.
- Výchozí povolení čerpadla true; před aktivací vypnout všechny Node-RED zápisy na Y04. Ohřev nadále řídí Node-RED, žádné povely 0–3 kW.
- Zdrojová verze připravená pro GitHub; aktivace na instalaci vyžaduje samostatné předání Y04. Publikace zdrojů sama řízení v HA nezapíná.

- TUV čte nový select stupně ohřevu a v hlavní kartě zobrazuje povolení či důvod blokace firmware.
- Diagnostika doplněna o X16, napájení čidel Y15 a dobu běhu Kicony.
- Horní teplota zůstává z externího zařízení; TUV2 se nečte. Obnova a porucha čidel skryjí odhad zásoby tepla.
- Starší nastavení získá nové entity při zachování vlastního mapování teplot a čerpadla. Ohřev se pouze čte, bez heartbeat nebo ovládání ohřevu.


## 0.6.2

- Odhad nabití TUV mezi horní a spodní teplotou podle dohodnutého vrstveného modelu; neplatná nebo zastaralá čidla zobrazí pomlčku. Podrobnosti výpočtu jsou v Diagnostice.
- Výkony sítě L1, L2 a L3 z Victron MQTT pod výkonem měničů, pouze čísla ve W se zachováním znaménka. Nové topics se doplní do starší konfigurace, vlastní a prázdné nastavení se zachová.
- Stejné rozměry tří horních karet, všechny bargrafy tlusté 18 px a nová ikona měničů.
- Pouze sledování, bez příkazů do zařízení.

## 0.6.1

- Výkon měničů čte potvrzený MQTT topic ConsumptionOnOutput/L1/Power, místo senzoru HA. Sdílí příjem a diagnostiku Victronu; bargraf zůstává do 7 kW.
- Nová položka mqtt_topics.inverter se doplní při načtení starší konfigurace; uložené ostatní topics se zachovají a výslovně prázdná položka zůstane vypnutá.
- Neplatná, uložená, zastaralá nebo nepřijatá MQTT zpráva se nevydává za živé měření. Bez náhrady hodnotou z HA.

## 0.6.0

- Kompaktní hlavní přehled bez úvodního textu, duplicitní karty baterie a simulátoru.
- Výrazný SOC, směr a výkon baterie, procentní bargraf SOC a výkonové bargrafy: výroba 8 kW, baterie 7 kW, měniče 7 kW.
- Výkon měničů ze sensor.vystupni_vykon, pouze čtení ve společném HA cyklu; W/kW, platnost a stáří se ověřují. Podrobnosti jsou v Diagnostice.

## 0.5.2

- Hlavní přehled bez diagnostických popisků, časových hlášení, zdrojových štítků a technických poznámek. Hodnoty, jejich názvy a výrazné provozní stavy zůstávají.
- Časy, platnost, připojení, vysvětlení údajů a verze jsou soustředěné do rozbalovací sekce Diagnostika.
- Neplatné a zastaralé hodnoty se stále nezobrazují jako platná čísla; podrobný důvod je v diagnostice.

## 0.5.1

- Výraznější stav TUV: samostatný nápis vypnutého nebo zvoleného ohřevu, kontrastní vybraný stupeň s fajfkou a jasně označené neaktivní stupně.
- Barevné a textové štítky čerpadla a vypnutého řízení doplňkem. Neznámý stupeň se odlišuje od vypnutého.
- Indikátory nadále zobrazují pouze hlášení HA, bez fyzických příkazů.

## 0.5.0

- Hlavní energetický přehled napojen na skutečnou výrobu, domácí baterii, TUV a Teslu. Bez nahrazování chybějících živých údajů simulací.
- Součet výroby pouze z platných dat všech tří stringů; rozlišení nuly, zastaralých údajů, chyb a výpadků jednotlivých zdrojů.
- TUV ukazuje obě teploty, právě jeden hlášený celkový stupeň 0/1/2/3 kW a čerpadlo. Tesla odděluje skutečný a nastavený proud.
- Simulátor, modelová doporučení, balancování a modelová historie přesunuty do samostatné rozbalovací sekce. Podrobnosti jednotlivých vstupů zůstávají v diagnostice.
- Oprava falešného zastarávání neměnných hodnot TUV/Tesly: pevná čtecí šablona HA získává aktuální last_reported přímo ze stavů místo uloženého JSON.
- Pouze sledování; žádné fyzické příkazy, probouzení auta ani změny živého Node-RED.

## 0.4.0

- Živá karta Tesly: SOC, kabel, stav nabíjení, výkon a skutečný proud, zvlášť nastavený proud.
- Čtení existujících entit HA ve společném snímku s TUV; žádné přímé volání Tessie, probouzení auta ani řízení nabíjení.
- Nastavitelné entity, rozlišení chybějících/neplatných/zastaralých dat, převod W/kW a zachování platné nuly.
- Živé hodnoty zůstávají oddělené od simulačních doporučení a historie.

## 0.3.0

- Čtení skutečných teplot TUV, čerpadla a celkového stupně ohřevu přes interní Home Assistant API, bez zadávání dalšího tokenu.
- Nová živá karta TUV oddělená od simulace; výkon je označen jako jmenovitý, příkon se neměří.
- Neurčený stupeň při chybějících nebo konfliktních stavech, kontrola stáří teplot a komunikace, obnova po výpadku.
- Pouze čtení, žádné fyzické příkazy ani změny Node-RED. Živá historie a živá doporučení zatím nejsou připojené.

## 0.2.0

- Čtecí adaptér Victron MQTT přes stávající broker a most: nastavitelné topics a samostatný panel skutečných dat.
- Rozlišení čekání na první zprávu, neplatných, zastaralých a retained údajů; restart ani odpojení neobnovují staré měření jako aktuální.
- Přístupové údaje pouze v backendové konfiguraci. Adaptér nic nepublikuje, fyzické řízení zůstává vypnuté a simulace oddělená.

- Světlý frontend inspirovaný evcc: barevný podíl solárních stringů, samostatné karty TUV a Tesly, výrazné hodnoty a mobilní rozložení.
- Karty oddělují simulovaný výkon a doporučení; při ztrátě spojení se aktuální hodnoty a doporučení skryjí.
- Solární výroba: aktuální simulovaný výkon stringů Střecha, Přístřešek a Plot a jejich celkový výkon ve W.
- Rozlišení nulové výroby, neplatného a zastaralého měření; noční scénář s nulovou výrobou.

## 0.1.1

- Vlastní ikona v hlavičce, záložce prohlížeče, mobilním zástupci, dokumentaci a prezentaci doplňku v Home Assistantu.

## 0.1.0

- První balení existujícího simulátoru pro instalaci přes repozitář Home Assistantu.
- Go backend, Vue/TypeScript přehled a SQLite historie s uchováním 7 dní.
- Scénáře SOC, teplotní blokace, balancování, neplatných dat a přetížení.
- Ingress bez veřejného portu, databáze v `/data`.
- Vydávání obrazů pro amd64/aarch64 s testem každé architektury a kontrolou anonymního stažení před aktualizací větve stable.
- Fyzické ovládání zůstává nedostupné; jde o simulaci v observe_only.
