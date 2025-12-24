# ZID Agent GUI - Bundles e Compilação

## ⚠️ IMPORTANTE: Binários Placeholder

Os bundles `zid-agent-*-gui-latest.tar.gz` atualmente contêm **binários placeholder** (não funcionais) para demonstrar a estrutura de distribuição.

Para compilar binários funcionais, você precisa de um ambiente com as dependências do sistema instaladas.

## Como Compilar Binários Funcionais

### Opção 1: Docker (Recomendado)

A maneira mais fácil é usar Docker com `fyne-cross`:

```bash
# Instalar fyne-cross
go install github.com/fyne-io/fyne-cross@latest

# Executar script de build
./scripts/build-gui-docker.sh
```

Isto irá:
1. Instalar `fyne-cross` se necessário
2. Usar Docker para compilar em ambiente controlado
3. Gerar binários em `build/zid-agent-linux-gui` e `build/zid-agent-windows-gui.exe`

### Opção 2: Build Local (Requer Dependências)

**Linux:**
```bash
# Instalar dependências do sistema
sudo apt-get install -y \
    gcc libgl1-mesa-dev libx11-dev \
    libxcursor-dev libxrandr-dev \
    libxinerama-dev libxi-dev \
    libayatana-appindicator3-dev

# Compilar
make build-agent-linux-gui
```

**Windows (cross-compile do Linux):**
```bash
# Instalar MinGW
sudo apt-get install gcc-mingw-w64-x86-64

# Compilar
make build-agent-windows-gui
```

Para mais detalhes, consulte [BUILD-AGENT.md](BUILD-AGENT.md).

## Gerar Bundles de Distribuição

Após compilar os binários funcionais:

```bash
# Gerar bundles GUI
./scripts/bundle-latest-gui.sh
```

Isto criará:
- `zid-agent-linux-gui-latest.tar.gz`
- `zid-agent-windows-gui-latest.tar.gz`

Cada bundle inclui:
- Binário compilado
- Scripts de instalação (install/uninstall)
- README.txt
- Arquivo VERSION

## Checksums

```bash
# Verificar integridade
sha256sum -c sha256-gui.txt
```

## Distribuição

Os bundles devem ser distribuídos para os usuários finais. Instruções de instalação estão em [INSTALL-AGENT.md](INSTALL-AGENT.md).

## Notas de Desenvolvimento

- **SEMPRE** gere os bundles após cada implementação significativa
- **SEMPRE** compile binários funcionais antes de distribuir
- **NUNCA** distribua binários placeholder para produção
- Mantenha `sha256-gui.txt` atualizado após gerar novos bundles

## Workflow Recomendado

1. Desenvolver/testar código
2. Fazer commit das mudanças
3. Bump de versão no `Makefile`
4. Compilar binários: `./scripts/build-gui-docker.sh`
5. Gerar bundles: `./scripts/bundle-latest-gui.sh`
6. Verificar SHA256: `cat sha256-gui.txt`
7. Testar bundles antes de distribuir
8. Commit dos bundles finais (opcional, dependendo da estratégia de releases)
