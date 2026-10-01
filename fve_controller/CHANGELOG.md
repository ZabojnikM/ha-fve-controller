# Changelog

## 0.1.0

- První balení existujícího simulátoru pro instalaci přes repozitář Home Assistantu.
- Go backend, Vue/TypeScript přehled a SQLite historie s uchováním 7 dní.
- Scénáře SOC, teplotní blokace, balancování, neplatných dat a přetížení.
- Ingress bez veřejného portu, databáze v `/data`.
- Vydávání obrazů pro amd64/aarch64 s testem každé architektury a kontrolou anonymního stažení před aktualizací větve stable.
- Fyzické ovládání zůstává nedostupné; jde o simulaci v observe_only.
