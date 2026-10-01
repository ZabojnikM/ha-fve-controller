# Architektura veřejného projektu

Existující implementace je Go HTTP server a rozhodovací jádro, Vue/TypeScript frontend a SQLite. Celý Docker build context leží v `fve_controller`. Frontend se sestaví v prvním stage, backend v druhém. Finální obraz obsahuje pouze binárku a statické assety.

Tato verze používá pouze modelové vstupy. Znaménko baterie: kladné nabíjení, záporné vybíjení. Hystereze, priority, balancování a kvalita dat jsou testované v Go. Frontend nerozhoduje o výkonu. Neexistuje fyzický transport příkazů ani API umožňující zapnout řízení.

Docker nastavuje DATA_DIR=/data. SQLite obsahuje nastavení (aktuálně dokončení simulovaného balancování) a modelovou historii. Po restartu se obnoví pouze tento záznam; starý požadavek se nikdy nepovažuje za potvrzené měření. Žádné nové provozní nastavení se tímto balením nezavádí.

Ingress poslouchá na 8099 a přijímá jen spojení od 172.30.32.2. Relativní URL assetů i API fungují pod cestou Ingressu. Lokální vývoj bez INGRESS_ONLY poslouchá ve výchozím stavu jen na 127.0.0.1. Doplněk nepotřebuje Supervisor API, HA API, host networking ani privilegovaný přístup.

Soukromé podklady a místní konfigurace nejsou publikované. Pro budoucí připojení instalace je nutné odděleně ověřit vstupy, zařízení a jejich limity. Nasazení řízení je samostatný krok vyžadující explicitní povolení.
