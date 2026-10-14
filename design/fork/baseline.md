# Baseline (Phase 1)

Stand: 2026-10-14. Der übernommene Code wird unverändert gebaut und getestet,
um festzuhalten, was schon vor der Modernisierung grün oder rot ist.

## Gegenstand

- Commit `880cb899b` (`main`, 2024-10-11). Das ist der letzte Upstream-Stand,
  `v0.19.0` plus 63 Commits.
- `go.mod`: `go 1.21`. Kopia ist über `replace` auf `alcionai/kopia@41471159`
  gepinnt.

## Umgebung

- Windows Server 2025, 4 vCPU, 16 GB RAM
- Go **1.21.13**, die Version, mit der die Original-CI aus `go-version-file`
  bauen würde, bezogen über `GOTOOLCHAIN=go1.21.13`
- Go **1.27.2**, die aktuelle stabile Version, als Vorschau auf Phase 2
- Die Original-CI lief auf `ubuntu-latest`. Eine Linux-Baseline steht noch
  aus (siehe unten).

## Befehle

Alle Befehle laufen in `src/`. Die Tests entsprechen dem Job `Unit-Test-Suite`
aus `ci.yml`, nur ohne `-failfast`, damit alle Fehler sichtbar werden.

```bash
go build ./...
go build -o corso .
go vet -tags testing ./...
CORSO_M365_TEST_USER_ID=foo CORSO_SECONDARY_M365_TEST_USER_ID=foo \
  go test -count=1 -tags testing -json -p 1 -timeout 30m ./...
```

## Ergebnis

| Schritt | Go 1.21.13 | Go 1.27.2 |
|---|---|---|
| `go build ./...` | ✅ | ✅ |
| `go build -o corso .` | ✅ (172 MB, `corso --version` ok) | ✅ |
| `go vet -tags testing ./...` | ✅ | ✅ ¹ |
| `go test` (Pakete) | 78 ✅ · 5 ❌ · 55 ohne Tests | 77 ✅ · 6 ❌ · 55 ohne Tests |
| `go test` (Tests inkl. Subtests) | 5619 ✅ · 28 ❌ · 116 übersprungen | 5616 ✅ · 31 ❌ · 116 übersprungen |

¹ Vet ist mit 1.27.2 nur deshalb sauber, weil `go.mod` noch die Sprachversion
1.21 deklariert. Neuere Prüfungen wie `printf` mit nicht-konstantem
Format-String greifen erst ab `go 1.24` in `go.mod`. Das wird in Phase 2
relevant.

Die 116 übersprungenen Tests sind die Integrationssuiten (`*IntgSuite`). Sie
laufen nur mit Test-Tenant und S3 (`CORSO_CI_TESTS` usw.).

## Fehlgeschlagene Tests

Alle Fehler sind Windows-spezifisch. Unter beiden Go-Versionen ist das Bild
gleich. Bei den zeitabhängigen Fällen schwankt von Lauf zu Lauf, welche
Subtests rot werden.

| Paket | Test | Ursache |
|---|---|---|
| `cli/utils` | `TestMakeAbsoluteFilePath/absolute_path` | Der Test erwartet den Unix-Pfad `/tmp/dir`, unter Windows liefert `filepath.Abs` aber `C:\tmp\dir`. |
| `internal/kopia` | `TestGetNewPathRefs/Not_Found_Due_To_Mod_Time` | Zwei aufeinanderfolgende `time.Now()` sind unter Windows gleich (grobe Uhrauflösung), deshalb unterscheidet sich die Mod-Time nicht. |
| `internal/m365/collection/drive` | `TestURLCacheUnitSuite/TestGetItemProperties/*/32_concurrent_reqs` (wechselnde Subtests) | `"t" is not greater than "t"`: gleiche Uhrauflösung |
| `internal/operations` | `TestBackupOperation_PersistResults/Completed`, `TestExportOperation_Export/Completed`, `TestRestoreOperation_PersistResults/Completed` | `"t" is not less than "t"`: gleiche Uhrauflösung |
| `pkg/services/m365/api` | `TestChatsInfo/Empty_chat` (nicht in jedem Lauf) | `Modified.After(start)` ist false: gleiche Uhrauflösung |
| `pkg/services/m365/api/graph` | `TestThrottlingMiddleware/*` | `0 < time.Since(start)` schlägt fehl, weil die gemessene Dauer 0 ist: gleiche Uhrauflösung |

Ein weiterer Fehler trat nur im ersten Lauf auf. `internal/converters/vcf`
(`TestConvert_contactable_to_vcf`) scheiterte an CRLF-Zeilenenden in den
eingebetteten Testdaten, weil Git unter Windows standardmässig mit
`core.autocrlf=true` auscheckt. Mit LF-Checkout ist der Test grün. Das Repo
hat keine `.gitattributes`-Regel für Zeilenenden, nur `*.svg binary`.

## Folgerungen

- Mit dem unveränderten Code gibt es **keine Build- und keine Vet-Fehler**,
  auch nicht mit Go 1.27.2.
- Unter Windows gibt es **keine Testfehler, die auf echte Defekte
  hindeuten**. Die roten Tests messen Zeitdifferenzen im Bereich der
  Uhrauflösung oder setzen Unix-Pfade voraus.
- Die eigentliche Referenz ist Linux, wie in der Original-CI. Die
  Linux-Baseline läuft über GitHub Actions im Fork, sobald dort ein
  Build- und Test-Workflow existiert (Phase 2 oder 6).
- Empfehlung für eine spätere Phase: `.gitattributes` mit
  `* text=auto eol=lf` ergänzen, damit Windows-Checkouts dieselben
  Testdaten wie Linux sehen.
