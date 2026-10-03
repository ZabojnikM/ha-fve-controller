# FVE Controller

<img src="icon.png" alt="Ikona FVE Controller" width="96" height="96">

Místní přehled baterie, TUV a Tesly pro Home Assistant. Go backend počítá doporučení, Vue/TypeScript zobrazuje živá data v hlavních kartách a samostatný simulátor. SQLite uchovává pouze modelovou historii a dokončení simulovaného balancování.

**Verze 0.5.0 zobrazuje v hlavním přehledu Victron přes MQTT a TUV i Teslu přes Home Assistant API a obsahuje oddělený rozbalovací simulátor.** `observe_only=true`, `control_enabled=false`; neposílá příkazy fyzickým zařízením. MQTT zapněte a nastavte v konfiguraci doplňku. Čtení TUV je výchozím nastavením zapnuté a používá interní přístup Supervisoru. Současný Node-RED zůstává beze změny.

Instalační repozitář: `https://github.com/ZabojnikM/ha-fve-controller#stable`.

Otevřete webové rozhraní přes Home Assistant Ingress. Přihlášení do GHCR ani publikovaný síťový port nejsou potřeba. Data jsou uložena v `/data/simulation.db` a přežijí restart i aktualizaci doplňku. Odinstalace může data odstranit; před aktualizací používejte zálohu HA.

Podrobný postup najdete v DOCS.md a změny verzí v CHANGELOG.md. Větev stable zveřejňuje architektury až po sestavení a spuštění odpovídajících obrazů v CI.
