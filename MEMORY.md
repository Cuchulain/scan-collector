# ISBN Collector: stav pro navázání

## Projekt
- Go HTTP receiver naslouchá na portu `8765`, přijímá POST JSON s poli `timestamp`, `content`, `format`, `deviceId` a zapisuje denní CSV `isbn-YYYY-MM-DD.csv` do `/data`.
- Docker Compose mapuje `./data:/data`; image se sestavuje z `Dockerfile`.
- Původní `isbn_receiver.py` byl nahrazen Go implementací.
- `go test ./...` prošlo.

## GitHub publikace
- Uživatel chce vytvořit soukromý GitHub repozitář přes `gh` a pushnout projekt.
- Poslední `gh auth status` potvrdil přihlášený účet `Cuchulain` na GitHub.com, HTTPS, scope `repo` a `workflow`.
- Pracovní adresář `/Users/merlin/Tools/isbn-collector` zatím nebyl Git repozitář (`git status` vrátil „Not a git repository“).
- Zatím nebyl vytvořen vzdálený repozitář a nic nebylo commitnuto ani pushnuto. Předchozí pokus byl uživatelem přerušen.
- Zamýšlený název repozitáře: `isbn-collector`, soukromý, pod účtem `Cuchulain`. Zamýšlený GHCR image: `ghcr.io/cuchulain/isbn-collector`.
- `.github/workflows/publish-image.yml` publikuje při pushi do `main` tagy `latest` a SHA commitu a vyžaduje, aby GitHub Actions směly zapisovat balíčky. Merges do `main` spustí workflow jako push.

## Navázání
1. Ověřit aktuální obsah pracovního adresáře a že `MEMORY.md` patří do commitu.
2. Inicializovat Git, nastavit `main`, vytvořit úvodní commit a přes `gh` vytvořit privátní repozitář `Cuchulain/isbn-collector` a pushnout obsah.
3. Ověřit remote a stav GitHub Actions; nevypsat ani nezapisovat žádné přihlašovací údaje.

Komunikovat s uživatelem česky.
