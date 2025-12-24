# Instruções de Compilação - ZID Agent GUI

## Situação Atual

Os bundles foram gerados com **binários placeholder** porque a compilação dos agents GUI requer:

1. **Docker rodando** com permissões para o usuário atual, OU
2. **Dependências do sistema** instaladas localmente

## Opção 1: Configurar Docker (Recomendado)

### Adicionar usuário ao grupo docker:

```bash
# Adicionar seu usuário ao grupo docker
sudo usermod -aG docker $USER

# Aplicar mudanças (ou fazer logout/login)
newgrp docker

# Verificar se funciona
docker ps
```

### Compilar com Docker:

```bash
cd /home/guilhermemulato/Nextcloud/Soul\ Solucoes/dev/zid-proxy

# Método 1: Usar script automatizado
./scripts/build-gui-docker.sh

# Método 2: Usar make target
make build-agent-fyne-cross
```

Depois de compilar, gerar os bundles:

```bash
./scripts/bundle-latest-gui.sh
```

## Opção 2: Compilar Localmente (Requer Dependências)

### Instalar dependências no Ubuntu/Debian:

```bash
sudo apt-get update
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

### Compilar:

```bash
cd /home/guilhermemulato/Nextcloud/Soul\ Solucoes/dev/zid-proxy

# Linux
make build-agent-linux-gui

# Windows (requer MinGW)
sudo apt-get install gcc-mingw-w64-x86-64
make build-agent-windows-gui
```

### Gerar bundles:

```bash
./scripts/bundle-latest-gui.sh
```

## Verificar Compilação

Após compilar com sucesso, verifique:

```bash
# Ver binários gerados
ls -lh build/zid-agent-*-gui*

# Testar Linux (se compilou localmente)
./build/zid-agent-linux-gui -version

# Gerar e verificar bundles
./scripts/bundle-latest-gui.sh
sha256sum zid-agent-*-gui-latest.tar.gz
```

## Status dos Bundles Atuais

⚠️ **IMPORTANTE**: Os bundles atuais (`zid-agent-*-gui-latest.tar.gz`) contêm binários placeholder que mostram uma mensagem de erro ao serem executados.

Para distribuir em produção:
1. Configure Docker conforme Opção 1
2. Compile os binários reais
3. Regere os bundles
4. Verifique os checksums

## Comandos Rápidos

```bash
# Setup Docker (uma vez)
sudo usermod -aG docker $USER
newgrp docker

# Compilar
./scripts/build-gui-docker.sh

# Empacotar
./scripts/bundle-latest-gui.sh

# Verificar
ls -lh *-gui-latest.tar.gz
sha256sum *-gui-latest.tar.gz
```

## Troubleshooting

**Erro: "permission denied while trying to connect to the docker API"**
- Solução: Adicione seu usuário ao grupo docker (ver Opção 1)

**Erro: "Package 'gl' was not found"** (compilação local)
- Solução: Instale as dependências do sistema (ver Opção 2)

**Erro: "MinGW compiler not found"**
- Solução: `sudo apt-get install gcc-mingw-w64-x86-64`

## Contato

Para dúvidas sobre compilação, consulte:
- BUILD-AGENT.md (detalhes técnicos)
- BUNDLES-README.md (processo de bundles)
