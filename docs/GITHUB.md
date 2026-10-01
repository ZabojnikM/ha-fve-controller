# Bezpečné vydávání

## Větve a verze

`main` obsahuje vývoj. Instalace v Home Assistantu používá výhradně URL `https://github.com/ZabojnikM/ha-fve-controller#stable`. `stable` neaktualizujte ručně. Větev není dostupná do prvního úspěšného vydání.

Verze `X.Y.Z` musí souhlasit v `fve_controller/config.yaml`, `fve_controller/web/package.json` a nadpisu CHANGELOG.md. Kontejner má tag `ghcr.io/zabojnikm/ha-fve-controller:X.Y.Z`, OCI label stejnou verzi a GitHub release/tag `vX.Y.Z`. Žádný `latest` se nepoužívá. Přesné digesty základních obrazů jsou v Dockerfile. Aktualizace těchto základů je kontrolovaná změna s novým buildem.

## Další aktualizace

1. Upravte zdroje na `main`, zvyšte všechny výše uvedené verze a doplňte changelog. Zkontrolujte soubory pomocí `git diff --cached` a `python3 scripts/check_publication.py`. Commitněte a pushněte běžným pushem bez force.
2. Nechte projít workflow **Check project**. Obě nativní úlohy sestaví frontend/backend a spustí hotový kontejner, otestují API, relativní assety, Ingress ACL a zachování SQLite při nahrazení kontejneru.
3. Spusťte **Release verified add-on** na větvi main se vstupem nové verze, např. `gh workflow run release.yml --ref main -f version=X.Y.Z`.
4. Workflow znovu sestaví a otestuje oba obrazy, nahraje neměnné tagy `X.Y.Z-amd64` a `X.Y.Z-aarch64`, pak vytvoří společný manifest `X.Y.Z`. Již existující tag s jiným zdrojovým commitem se nepřepisuje.
5. Následuje anonymní stažení manifestů a všech vrstev. Bez tohoto úspěchu se stable ani GitHub release nezmění.
6. Teprve potom workflow provede běžný fast-forward push stable a tagu `vX.Y.Z` v jedné atomické operaci, následně zveřejní GitHub Release. Obsah stable je přesně ověřený strom z main. Další vydání zachovávají oba rodiče historie, bez force push.
7. V HA obnovte obchod/zkontrolujte aktualizace, vytvořte zálohu a zvolte Aktualizovat. `/data` zůstává zachováno. Ověřte log a označení SIMULACE.

## První zveřejnění GHCR

GHCR vytváří nový package standardně jako private i u veřejného repozitáře. Po prvním uploadu musí vlastník v GitHub profilu → Packages → ha-fve-controller → Package settings → Change visibility nastavit **Public**. Repozitář má právo publikovat přes vestavěný GITHUB_TOKEN (`packages: write`), token se neukládá do projektu. Nastavení veřejnosti je jednorázové, nikoli změna viditelnosti repozitáře.

Pokud první release skončí na anonymní kontrole, po nastavení Public zopakujte pouze neúspěšné úlohy. Stejné tagy ze stejného SHA lze znovu použít. Anonymní kontrola je samostatně spustitelná: `python3 scripts/verify_registry.py ghcr.io/zabojnikm/ha-fve-controller X.Y.Z`.

## Selhání a návrat

Selže-li build, test nebo veřejný přístup, HA dál vidí předchozí stable. Při chybě po uploadu neměňte obsah stejné verze: oprava zdrojů potřebuje nové číslo. Selhání vytvoření GitHub Release po atomickém pushi lze obnovit opakováním workflow stejného SHA. Pokud už se main posunul, nejprve dokončete vydání existujícího tagu ručně (`gh release create vX.Y.Z --verify-tag --notes-file fve_controller/CHANGELOG.md`) a nepřepisujte jej.

Rollback provádějte přes HA zálohu nebo nové vyšší vydání s opraveným/revertovaným obsahem. Nesnižujte verzi stable, nemažte používané GHCR tagy a nepřepisujte historii.

## Kontrola soukromí a oprávnění

Před prvním zveřejněním nebyla v pracovní složce žádná Git historie ani remote. Původní soukromé podklady zůstávají na disku a mimo index. Automatická kontrola prochází sledované soubory, nikoli jen .gitignore. Pokud se tajný údaj někdy dostane do historie, publikaci zastavte, údaj zneplatněte a dohodněte nápravu; nepřepisujte historii bez souhlasu.

Workflow používá jen nezbytná oprávnění: kontroly contents:read, upload packages:write, závěrečné vydání contents:write. Žádné PAT v secrets projektu není nutné.

## Oficiální podklady ověřené 1. 10. 2026

- [HA repozitáře](https://developers.home-assistant.io/docs/apps/repository/)
- [Publikování obrazů a multiarch manifest](https://developers.home-assistant.io/docs/apps/publishing/)
- [Ingress a prezentace](https://developers.home-assistant.io/docs/apps/presentation/)
- [Konfigurace doplňku](https://developers.home-assistant.io/docs/apps/configuration/)
- [GHCR autentizace a veřejnost](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
