# skills-manager

CLI em Go para selecionar skills de um catálogo local e disponibilizá-las no projeto por links simbólicos em `.agents/skills`. O catálogo continua sendo a fonte de verdade: alterações feitas na origem aparecem nos projetos vinculados sem reinstalação.

## Instalação

Em macOS e Linux, a instalação da release mais recente pode ser iniciada com:

```sh
curl -fsSL https://github.com/danielpavone/skills-manager/releases/latest/download/install.sh | sh
```

No Windows PowerShell:

```powershell
curl.exe -fsSL https://github.com/danielpavone/skills-manager/releases/latest/download/install.ps1 | powershell -NoProfile -Command -
```

Os instaladores baixam o arquivo compactado e `skills-manager_checksums.txt` por HTTPS, conferem o SHA-256 antes de extrair e só substituem o binário depois da verificação. Para uma instalação manual, baixe ambos os arquivos da release, confira o checksum com `sha256sum` ou `Get-FileHash` e extraia o arquivo compatível com seu sistema.

Os instaladores aceitam `SKILLS_MANAGER_REPO`, `SKILLS_MANAGER_VERSION`, `SKILLS_MANAGER_INSTALL_DIR` e `SKILLS_MANAGER_DOWNLOAD_BASE_URL` para releases alternativas. A URL-base deve ser HTTPS fora dos testes.

## Configuração e uso

Informe a pasta do repositório de skills. A CLI procura automaticamente o catálogo em `.agents/skills`, onde cada pasta de skill precisa de um `SKILL.md` legível:

```text
~/skills-repository/
└── .agents/skills/
    ├── code-review/SKILL.md
    └── tdd/SKILL.md
```

Configure ou consulte o catálogo:

```sh
skills-manager config set ~/skills-repository
skills-manager config show
```

Para compatibilidade, também é possível informar diretamente o caminho completo de `.agents/skills` ou outro diretório que já contenha as pastas das skills.

Na raiz do projeto, execute `skills-manager` para abrir a TUI. A confirmação cria apenas os links simbólicos selecionados; links corretos são preservados, conflitos não são sobrescritos e a remoção exclui somente o vínculo local.

Atalhos principais:

| Tecla | Ação |
| --- | --- |
| `/` | pesquisar pelo nome |
| `↑`/`↓`, `j`/`k` | percorrer a lista |
| `space` | selecionar ou desmarcar |
| `enter` | abrir confirmação |
| `y` | aplicar alterações |
| `esc` | voltar ou cancelar |
| `q`, `ctrl+c` | sair |

`--version` exibe a versão compilada. A CLI retorna `0` em sucesso ou cancelamento explícito, `1` quando há falha de configuração ou operação e `2` para uso inválido.

## Compatibilidade

As releases incluem Linux, macOS e Windows em `amd64` e `arm64`. Linux e macOS usam `tar.gz`; Windows usa `zip`. A criação de links simbólicos depende das permissões do sistema. Se o Windows negar a operação, habilite o Developer Mode ou execute em um contexto com privilégio adequado; a CLI reportará a skill afetada e não copiará o conteúdo como fallback.

## Desenvolvimento

Requer Go 1.25 ou superior. Testes, integração e cenários de filesystem controlados executam com:

```sh
go test ./...
```

Os fixtures dos instaladores podem ser executados separadamente em sistemas POSIX:

```sh
sh scripts/install_test.sh
```

Em Windows, execute `scripts/install.tests.ps1` no PowerShell. O workflow de CI executa esses testes e `go test ./...` em runners Linux, macOS e Windows. Releases são publicadas pelo GoReleaser quando uma tag `v*` é enviada.
