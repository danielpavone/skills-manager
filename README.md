# skills-manager

CLI em Go para selecionar skills de um catálogo local e disponibilizá-las no projeto por links simbólicos em `.agents/skills`, `.claude/skills` ou `.devin/skills`. O catálogo continua sendo a fonte de verdade: alterações feitas na origem aparecem nos projetos vinculados sem reinstalação.

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

## Configuração inicial

Informe a raiz do repositório de skills com `config set`. Cada skill deve ser uma pasta contendo um `SKILL.md` legível:

```text
~/skills-repository/
└── .agents/skills/
    ├── code-review/SKILL.md
    └── tdd/SKILL.md
```

Sem opções adicionais, a CLI procura o catálogo em `<repositório>/.agents/skills` e instala os links em `<projeto>/.agents/skills`:

```sh
skills-manager config set ~/skills-repository
```

Use `--target` no setup para escolher outra convenção. Um destino fica ativo por vez:

| Opção | Catálogo procurado a partir da raiz | Destino no projeto |
| --- | --- | --- |
| omitida ou `--target .agents` | `.agents/skills` | `.agents/skills` |
| `--target .claude` | `.claude/skills` | `.claude/skills` |
| `--target .devin` | `.devin/skills` | `.devin/skills` |

```sh
skills-manager config set ~/skills-repository --target .claude
skills-manager config set ~/skills-repository --target .devin
```

Também é possível informar diretamente uma pasta que já contenha as skills. Nesse caso, o caminho é usado como catálogo, enquanto `--target` continua definindo onde os links serão instalados no projeto:

```sh
skills-manager config set ~/skills-repository/catalog --target .claude
```

Consulte a configuração ativa com:

```sh
skills-manager config show
```

Exemplo de saída:

```text
catálogo: /Users/alice/skills-repository/.claude/skills
destino: .claude/skills
```

Configurações criadas por versões anteriores, sem um destino salvo, continuam procurando e instalando em `.agents/skills`.

## Gerenciando as skills do projeto

Na raiz do projeto, execute `skills-manager` para abrir a TUI. A confirmação cria os links no destino configurado; links corretos são preservados, conflitos não são sobrescritos e a remoção exclui somente o vínculo local.

```sh
cd ~/projects/my-project
skills-manager
```

Atalhos principais:

| Tecla | Ação |
| --- | --- |
| `/` | pesquisar pelo nome |
| `↑`/`↓`, `j`/`k` | percorrer a lista |
| `space` | selecionar ou desmarcar |
| `enter` | aplicar a pesquisa; abrir ou concluir a confirmação; voltar às skills após aplicar |
| `y` | aplicar alterações |
| `esc` | cancelar ou limpar a pesquisa; voltar ou cancelar a operação |
| `q`, `ctrl+c` | sair |

Durante a pesquisa, `enter` aplica o filtro e retorna à navegação. Depois de selecionar uma skill filtrada, pressione `esc` para limpar a pesquisa e voltar à lista completa; as seleções já feitas são preservadas.

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
