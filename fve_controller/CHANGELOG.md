# Changelog

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
