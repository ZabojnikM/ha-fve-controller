# Changelog

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
