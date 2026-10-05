# Řízení čerpadla TUV Y04

Zdrojová verze 0.7.0 připravená pro GitHub 4. 10. 2026. Aktivace na instalaci je samostatný krok; publikace zdrojů ji neprovádí.

**Před aktivací doplňku s `pump_control_enabled: true` vypněte všechny Node-RED větve zapisující na Y04, včetně ručních a časových povelů.** Totéž platí pro další HA automatizace zapisující na tento výstup. Y04 musí mít jediného správce.

Výchozí konfigurace doplňku má `pump_control_enabled: true`: je připravená k funkčnímu převzetí čerpadla nezávisle na ohřevu. Výslovně uložené `false` se zachová. Lokální běh bez options.json zůstává bez řízení; existující options bez této položky používají true. Pro dočasné sledování nastavte false a restartujte doplněk před jeho spuštěním proti HA.

## Automatika a ruční režim

Backend vyhodnocuje požadavky každou sekundu bez otevřeného prohlížeče. Horní teplota pochází ze stávajícího externího čidla, spodní z TUV1. Zapnutí: horní >58 °C a spodní <57 °C. Vypnutí: horní <57 °C nebo spodní >=57 °C. V ostatním hysterézním pásmu zachová procesní požadavek; při startu automatiky jej odvodí z živého HA stavu, nikoli uloženého povelu.

Denně v 18:30 Europe/Prague (včetně změn letního času) zahájí servisní protočení. Jeho 30 sekund se počítá od hlášení zapnutého čerpadla z HA. Výsledný požadavek je procesní míchání OR servisní protočení. Konec servisního požadavku tedy nevypne potřebné procesní míchání. Datum se uloží do SQLite před zahájením; tentýž den se po restartu neopakuje, časovač se neobnovuje a zmeškané protočení se po 18:31 nedohání. Chyba zápisu zabrání zahájení servisu.

Centrální volba **Automatika TUV / Ruční ovládání** nyní řídí pouze čerpadlo. Ruční režim zastaví procesní i servisní automatiku a zpřístupní Čerpadlo Zap/Vyp. Přechod do ručního režimu začíná požadavkem Vyp. Režim se ukládá, ruční povel nikoli: po restartu v ručním režimu začíná požadavek Vyp a čeká se na živý stav. Žádný starý ruční povel se neobnovuje jako potvrzení.

Ohřev 0–3 kW zůstává výhradně v Node-RED. Neexistuje aktivní transport ani API pro ohřev, heartbeat, odblokování, reset, ATS, wallbox nebo Teslu. Stupně ohřevu v dashboardu jsou pouze hlášený stav.

## Příkazy a potvrzení

Samostatný adaptér používá pouze switch.turn_on / switch.turn_off pro `ha_entities.pump` (výchozí Y04). Obecné HA service API není vystavené. Dva pevné POST endpointy `/api/tuv/mode` a `/api/tuv/pump` vyžadují Ingress, JSON, platný omezený obsah a token X-FVE-TUV získaný ze snímku. Token je ochrana požadavků dashboardu, nikoli Supervisor token; nové spuštění serveru jej mění.

Úspěšná odpověď služby není potvrzení. Dashboard odděluje požadovaný, odeslaný a HA potvrzený stav; potvrzení povelu vyžaduje následující snímek s očekávanou hodnotou. Čekání trvá nejvýše 15 s, po chybě/timeoutu je odstup 10 s. Nový opačný požadavek přeruší čekání na starý. Hlášení HA není důkaz průtoku a optimistická integrace nemusí potvrzovat fyzické relé; ověřte skutečnou zpětnou vazbu při předání.

Zapnutí včetně ručního vyžaduje platné dostupné teploty 0–100 °C bez limitu stáří hlášení, úspěšně načtený HA snímek bez časového limitu, platný stav čerpadla a vypnutý reset napájení čidel. Výpadek či neplatná data zruší požadavek Zap, při dostupné komunikaci se požaduje Vyp. Nedostupnost ani chyba povelu se nezobrazuje jako potvrzené vypnutí. Při řádném ukončení se doplněk pokusí poslat Vyp; výpadek napájení nebo násilné ukončení tento pokus nezaručuje. Místní watchdog Y04 není doložený.

## Předání na instalaci

1. Před nasazením ověřte typ instalace HA, architekturu, mapování Y04 a skutečnou stavovou odezvu.
2. Vypněte všechny Node-RED větve a jiné automatizace zapisující na Y04. Ohřev v Node-RED ponechte.
3. Teprve potom aktivujte doplněk s povoleným čerpadlem a ověřte ruční Zap/Vyp, potvrzení HA, režimy, výpadek a restart.
4. Při návratu nejdříve vypněte řízení čerpadla v doplňku a až potom obnovte starého správce.

Zdroje ověřeny 4. 10. 2026: [komunikace apps](https://developers.home-assistant.io/docs/apps/communication/), [Ingress](https://developers.home-assistant.io/docs/apps/presentation/), [Supervisor API](https://developers.home-assistant.io/docs/api/supervisor/endpoints/), [MQTT switch a optimistický stav](https://www.home-assistant.io/integrations/switch.mqtt/).
