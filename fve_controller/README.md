# FVE Controller

<img src="icon.png" alt="Ikona FVE Controller" width="96" height="96">

Místní přehled baterie, TUV a Tesly pro Home Assistant. Go backend počítá doporučení, Vue/TypeScript zobrazuje modelové vstupy a SQLite uchovává historii a dokončení balancování.

**Verze 0.1.1 je výhradně simulátor.** `observe_only=true`, `control_enabled=false`; neobsahuje klienty ani příkazy pro fyzická zařízení. Současný Node-RED může zůstat beze změny.

Instalační repozitář: `https://github.com/ZabojnikM/ha-fve-controller#stable`.

Otevřete webové rozhraní přes Home Assistant Ingress. Přihlášení do GHCR ani publikovaný síťový port nejsou potřeba. Data jsou uložena v `/data/simulation.db` a přežijí restart i aktualizaci doplňku. Odinstalace může data odstranit; před aktualizací používejte zálohu HA.

Podrobný postup najdete v DOCS.md a změny verzí v CHANGELOG.md. Větev stable zveřejňuje architektury až po sestavení a spuštění odpovídajících obrazů v CI.
