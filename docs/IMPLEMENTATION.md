# Implementační plán a rozhodnutí

## První milník
1. Go jádro s explicitní platností a stářím dat; deterministické testy ochran, SOC hystereze a společného rozpočtu.
2. Simulátor jako jediný vstup. Doporučení se nikdy neposílají zařízením. Žádný klient HA, MQTT ani Tessie, žádné přístupové tokeny.
3. SQLite: pouze simulovaná historie (7 dní) a potvrzené dokončení simulovaného balancování. Po restartu nové vstupy, žádné obnovené potvrzení výkonu.
4. Vue/TypeScript přehled se scénáři, důvody rozhodnutí a oddělením měřeného/požadovaného/odeslaného výkonu.
5. Docker + HA Ingress; build, testy a vizuální kontrola. Nasazení do HA je samostatný krok po zjištění typu instalace a CPU.

## Zjištění z podkladů
Prozkoumány všechny tři exporty (103, 46 a 186 uzlů), dashboard a starší analýza. Nepřenášet identifikátory, adresy ani exporty do aplikace. Staré flow nahrazuje některé nuly výchozí hodnotou pomocí `||`; nové jádro nulu zachovává. Staré výkonové prahy mají mezery. Export potvrzuje kritické hranice 10/13 a ochrany 16/17/20. Tesla v exportu používá 6 A, zadání má přednost: 5 A. Kapacity 17 a 25 kWh jsou rozporné a nejsou použity.

Časovače obsahují NT 19:00, 05:00 i duplicitní 22:00; VT 00:00 a 08:00. Kandidát NT 05–08 a 19–24 vyžaduje potvrzení instalace. Plánovač tarifu/predikce ani ATS sekvence se v prvním milníku neaktivují. UI uvádí plánování jako nepřipojené.

SOC vysoký vstup >=95, výstup <=93. Běžný cíl 0 W, vysoký SOC −300 W, balancování +300 W. TUV provozní cíl 60 °C, bezpečnostní mez >65 °C. Demo limit společných spotřebičů 4600 W je výhradně simulační parametr, nikoli limit instalace. Při omezeném MPPT a nulovém toku dovolí vysoký SOC jeden průzkumný krok TUV 1 kW; jde o výslovnou kvantizaci cíle −300 W pro diskrétní spirálu. Další zvýšení je možné až po čerstvé odezvě a prodlevě. Při balancování se průzkumný krok nepoužije.

## Ověřená oficiální dokumentace (1. 10. 2026)
- [Konfigurace apps](https://developers.home-assistant.io/docs/apps/configuration/): samostatný kontejner, architektury, /data, Ingress.
- [Ingress](https://developers.home-assistant.io/docs/apps/presentation/): port 8099, povolit pouze peer 172.30.32.2, relativní assety/API.
- [Komunikace](https://developers.home-assistant.io/docs/apps/communication/) a [Supervisor API](https://developers.home-assistant.io/docs/api/supervisor/endpoints/): token pouze na backendu; první verze žádná tato oprávnění nepotřebuje.
- [Instalace](https://www.home-assistant.io/installation/): aktuálně OS a Container; Container nemá apps. Historické označení Supervised není potvrzením podpory.
- [MQTT](https://www.home-assistant.io/integrations/mqtt/): broker a autentizace se ověřují samostatně; pozdější čtecí adaptér bude oddělen od publish/řízení.

## Další milníky
Pouze čtení skutečných vstupů a porovnání s Node-RED; ověření znamének, časů a potvrzení. Potom návrh jediného vlastníka každého výstupu a předání řízení po jednotlivých zařízeních s explicitním souhlasem a vlastním nasazením. Neověřený rozsah 16 A, kapacita, topics a ATS brání aktivnímu řízení, nikoli simulaci.
