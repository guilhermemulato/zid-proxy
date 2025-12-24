# ZID Agent - Guia de Instalação

O **ZID Agent** é uma aplicação com interface gráfica que roda na bandeja do sistema (system tray) e envia informações de identificação do computador (hostname e usuário) para o pfSense onde está o zid-proxy.

## Visão Geral

### O que o Agent faz?

- Roda silenciosamente em background com ícone na system tray
- Envia "heartbeat" a cada 30 segundos para o pfSense com:
  - Hostname da máquina
  - Nome do usuário logado
  - Versão do agent
- Descobre automaticamente o pfSense (via gateway default ou DNS)
- Exibe logs em janela dedicada (menu: clique direito no ícone → Logs)

### Por que instalar?

O agent enriquece a visualização de IPs ativos no pfSense, mostrando qual máquina e usuário está gerando tráfego, facilitando o monitoramento e troubleshooting.

---

## Instalação - Windows

### Requisitos

- Windows 10 ou superior
- Rede conectada ao pfSense (gateway padrão ou DNS `zid-proxy.lan`)

### Passo a Passo

1. **Baixar o agent:**
   - Baixe `zid-agent-windows-gui-latest.tar.gz`
   - Extraia o arquivo

2. **Instalar:**
   - Clique com botão direito em `install-windows.bat`
   - Selecione **"Executar como administrador"** (opcional, mas recomendado)
   - Siga as instruções na tela
   - O instalador irá:
     - Copiar o agent para `%LOCALAPPDATA%\ZIDAgent`
     - Criar atalho na pasta de inicialização
     - Opcionalmente iniciar o agent

3. **Verificar instalação:**
   - Procure o ícone **ZID** na bandeja do sistema (system tray)
   - Clique com botão direito → **Logs** para ver a conexão com pfSense

### Desinstalar (Windows)

Execute `uninstall-windows.bat` (como administrador se instalou como admin)

---

## Instalação - Linux

### Requisitos

- Ubuntu 20.04+ / Debian 11+ / Fedora 35+ ou similar
- Ambiente gráfico (X11 ou Wayland)
- System tray suportado:
  - **GNOME:** Instale extensão AppIndicator
    ```bash
    sudo apt install gnome-shell-extension-appindicator
    ```
  - **KDE/XFCE/MATE:** Suporte nativo

### Dependências do Sistema

**Ubuntu/Debian:**
```bash
sudo apt-get install -y \
    libgl1-mesa-dev \
    libx11-dev \
    libayatana-appindicator3-dev
```

**Fedora:**
```bash
sudo dnf install -y \
    mesa-libGL-devel \
    libX11-devel \
    libayatana-appindicator-devel
```

### Passo a Passo

1. **Baixar o agent:**
   ```bash
   wget https://seu-servidor/zid-agent-linux-gui-latest.tar.gz
   tar -xzf zid-agent-linux-gui-latest.tar.gz
   cd zid-agent-linux-gui
   ```

2. **Instalar:**
   ```bash
   ./install-linux.sh
   ```

   O instalador perguntará o método de instalação:
   - **Opção 1 (recomendado):** Systemd user service
   - **Opção 2:** XDG autostart

3. **Verificar instalação:**

   **Se instalou com systemd:**
   ```bash
   systemctl --user status zid-agent
   journalctl --user -u zid-agent -f
   ```

   **Opcional (iniciar junto com o boot, antes do login):**
   - O `install-linux.sh` oferece habilitar `systemd linger` para o seu usuário.
   - Observação: o ícone do tray só aparece após você fazer login na sessão desktop.

   **Se instalou com XDG autostart:**
   - Procure o ícone **ZID** na system tray
   - Clique com botão direito → **Logs**

### Desinstalar (Linux)

```bash
./uninstall-linux.sh
```

---

## Uso

### Menu do System Tray

Clique com botão direito no ícone ZID:

- **Logs:** Abre janela com histórico de mensagens (últimas 500)
- **Sair:** Encerra o agent

### Logs

A janela de logs mostra:
- Startup/shutdown do agent
- Descoberta do pfSense (IP do gateway ou DNS)
- Status de cada heartbeat (sucesso ou falha)

Exemplo:
```
2025-12-23 10:15:32 | ZID Agent v1.1.0 starting...
2025-12-23 10:15:32 | Heartbeat service started (hostname: DESKTOP-ABC, user: joao)
2025-12-23 10:15:33 | Heartbeat OK: 192.168.1.1
2025-12-23 10:16:03 | Heartbeat OK: 192.168.1.1
```

### Troubleshooting

**Ícone não aparece na tray:**
- **GNOME:** Instale `gnome-shell-extension-appindicator`
- **Wayland:** Teste em sessão X11
- Verifique se o agent está rodando: `ps aux | grep zid-agent`

**Heartbeat falha:**
- Verifique conectividade com pfSense: `ping <gateway>`
- Confirme que o pfSense está escutando na porta 18443
- Verifique firewall local (libere porta 18443 outbound)

**Ver logs detalhados (Linux systemd):**
```bash
journalctl --user -u zid-agent -f
```

**Ver logs detalhados (Windows):**
- Abra a janela de Logs pelo menu da tray
- Ou execute via linha de comando:
  ```cmd
  "%LOCALAPPDATA%\ZIDAgent\zid-agent.exe"
  ```

---

## Configuração no pfSense

Para que o agent funcione corretamente, o pfSense deve estar configurado:

1. Acesse **Services > ZID Proxy > Agent**
2. Habilite **Agent Listener**
3. Configure **Identity TTL** (padrão: 300s = 5 minutos)
4. Salve e aplique

O TTL define quanto tempo o pfSense mantém a identidade (machine/user) após o último heartbeat. Se o agent parar de enviar heartbeats, após o TTL a identidade é removida.

---

## Atualizações

### Windows

**Método Recomendado (Usando o Script de Atualização):**

1. Baixe a nova versão: `zid-agent-windows-gui-latest.tar.gz`
2. Extraia o arquivo
3. Execute `update-windows.bat` (clique duplo)

O script automaticamente:
- Para o agent em execução
- Substitui o binário
- Reinicia o agent

**Método Alternativo (Reinstalação Completa):**

1. Execute `uninstall-windows.bat`
2. Execute `install-windows.bat` da nova versão

### Linux

**Método Recomendado (Usando o Script de Atualização):**

```bash
# 1. Baixar e extrair
wget https://seu-servidor/zid-agent-linux-gui-latest.tar.gz
tar -xzf zid-agent-linux-gui-latest.tar.gz
cd zid-agent-linux-gui

# 2. Executar o updater
./update-linux.sh
```

O script automaticamente:
- Detecta se está instalado via systemd ou XDG
- Para o agent em execução
- Substitui o binário
- Reinicia o agent

**Método Alternativo (Manual):**

```bash
# Parar o serviço
systemctl --user stop zid-agent

# Substituir o binário
sudo cp zid-agent-linux-gui /usr/local/bin/zid-agent
sudo chmod +x /usr/local/bin/zid-agent

# Reiniciar o serviço
systemctl --user start zid-agent

# Verificar status
systemctl --user status zid-agent
```

**Método Alternativo (Reinstalação Completa):**

```bash
./uninstall-linux.sh
./install-linux.sh
```

---

## Perguntas Frequentes

**P: O agent consome muitos recursos?**
R: Não. O agent usa ~10-20MB de RAM e CPU desprezível (envia 1 requisição a cada 30s).

**P: Precisa rodar como administrador?**
R: Não. O agent funciona perfeitamente como usuário normal.

**P: Posso rodar vários agents na mesma rede?**
R: Sim! Cada máquina deve ter seu próprio agent. O pfSense rastreia por IP.

**P: O agent funciona em VPN?**
R: Sim, desde que a VPN permita acesso ao pfSense (gateway ou DNS).

**P: Como sei se está funcionando?**
R: Abra a aba **Active IPs** no pfSense e procure seu IP. Deve mostrar hostname e username.

**P: Posso desabilitar o autostart?**
R:
- **Windows:** Remova o atalho de `%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup`
- **Linux (systemd):** `systemctl --user disable zid-agent`
- **Linux (boot sem login):** `loginctl disable-linger $USER` (se você habilitou linger)
- **Linux (XDG):** Delete `~/.config/autostart/zid-agent.desktop`

---

## Suporte

Para problemas ou dúvidas:
1. Verifique os logs (menu Logs ou journalctl)
2. Consulte BUILD-AGENT.md para detalhes técnicos
3. Abra issue no repositório: https://github.com/guilherme/zid-proxy/issues
