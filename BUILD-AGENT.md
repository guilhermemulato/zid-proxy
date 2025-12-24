# Building zid-agent with GUI

Este documento descreve como fazer o build do `zid-agent` com interface gráfica (system tray).

## Dependências do Sistema

### Linux (Ubuntu/Debian)

```bash
# Instalar dependências necessárias para Fyne + systray
sudo apt-get install -y \
    gcc \
    libgl1-mesa-dev \
    libx11-dev \
    libxcursor-dev \
    libxrandr-dev \
    libxinerama-dev \
    libxi-dev \
    libxxf86vm-dev \
    libayatana-appindicator3-dev \
    pkg-config
```

### Fedora/RHEL

```bash
sudo dnf install -y \
    gcc \
    mesa-libGL-devel \
    libX11-devel \
    libXcursor-devel \
    libXrandr-devel \
    libXinerama-devel \
    libXi-devel \
    libayatana-appindicator-devel \
    pkg-config
```

### Windows

No Windows, você precisará de:
- GCC (via MinGW-w64 ou TDM-GCC)
- Recomendado: usar Docker ou cross-compile do Linux

### macOS

```bash
brew install pkg-config
```

## Build Commands

### Linux (nativo)

```bash
# Build local para testes
go build -o build/zid-agent-linux ./cmd/zid-agent

# Build com otimizações
go build -ldflags="-s -w -X main.Version=1.1.0" \
  -o build/zid-agent-linux ./cmd/zid-agent
```

### Windows (cross-compile do Linux)

```bash
# Requer configuração de cross-compile com MinGW
CGO_ENABLED=1 \
GOOS=windows \
GOARCH=amd64 \
CC=x86_64-w64-mingw32-gcc \
go build -ldflags="-s -w -H windowsgui -X main.Version=1.1.0" \
  -o build/zid-agent-windows.exe ./cmd/zid-agent
```

**Nota:** `-H windowsgui` esconde a janela de console no Windows.

### Usando fyne-cross (recomendado para cross-compile)

```bash
# Instalar fyne-cross
go install github.com/fyne-io/fyne-cross@latest

# Build Linux
fyne-cross linux -arch=amd64 -app-id=com.soulsolucoes.zidagent ./cmd/zid-agent

# Build Windows
fyne-cross windows -arch=amd64 -app-id=com.soulsolucoes.zidagent ./cmd/zid-agent
```

## Problemas Conhecidos

### 1. CGO Required

Tanto Fyne quanto systray requerem CGO. Isso significa:
- `CGO_ENABLED=1` (default em builds nativos)
- Dependências do sistema devem estar instaladas
- Cross-compile é mais complexo

### 2. System Tray no Wayland

Em ambientes Wayland (Ubuntu 22.04+), o system tray pode não funcionar perfeitamente.
Soluções:
- Instalar extensão AppIndicator no GNOME
- Usar XWayland
- Testar em ambiente X11

### 3. Fyne Dependency Size

Fyne adiciona ~20MB ao binário. Isso é normal para um toolkit GUI.

## Testes Rápidos (Sem GUI)

Se você quer apenas testar a lógica sem GUI:

```bash
# Rodar testes unitários (não requerem GUI)
go test ./internal/agentui/...

# Build apenas das bibliotecas internas
go build ./internal/agentui/...
```

## Build Flags Úteis

```bash
# Mostrar versão
./build/zid-agent-linux -version

# Build com informações de debug
go build -gcflags="all=-N -l" -o build/zid-agent-debug ./cmd/zid-agent

# Build estático (minimizar dependências)
CGO_ENABLED=1 go build -ldflags="-s -w -linkmode external -extldflags -static" \
  -o build/zid-agent-static ./cmd/zid-agent
```

## Makefile Targets (em desenvolvimento)

```bash
# Build GUI para Linux
make build-agent-linux-gui

# Build GUI para Windows (cross-compile)
make build-agent-windows-gui

# Bundle completo (pfSense + agents)
make bundle-latest
```

## Recomendações

Para desenvolvimento:
1. Use Linux nativo para builds de teste
2. Use `fyne-cross` para builds de produção multiplataforma
3. Teste system tray em ambiente X11 antes de Wayland

Para produção:
1. Use CI/CD com Docker images que têm as dependências
2. Gere bundles separados por plataforma
3. Assine binários Windows (futuramente)
