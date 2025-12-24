# Plano de Implementação: zid-agent com GUI (System Tray)

## ✅ STATUS: FASE 3 EM ANDAMENTO (Polimento Pós-MVP)

**Data:** 2025-12-23
**Progresso:** Fase 1 e 2 completas + Fase 3 iniciada
**Próximo passo:** Validar em campo (Windows/Linux) e ajustar UX conforme feedback

### Arquivos Criados/Modificados (Fase 1 + 2):

**Código (Fase 1):**
- ✅ `internal/agentui/ringbuffer.go` - Ring buffer genérico thread-safe
- ✅ `internal/agentui/logger.go` - Log manager com subscribers
- ✅ `internal/agentui/ringbuffer_test.go` - Testes do ring buffer
- ✅ `internal/agentui/logger_test.go` - Testes do log manager
- ✅ `cmd/zid-agent/heartbeat.go` - Loop de heartbeat isolado
- ✅ `cmd/zid-agent/tray.go` - System tray manager
- ✅ `cmd/zid-agent/logsui.go` - Janela de logs (Fyne)
- ✅ `cmd/zid-agent/assets/icon.go` - Ícone embedded
- ✅ `cmd/zid-agent/main.go` - Refatorado para GUI

**Build e Distribuição (Fase 2):**
- ✅ `Makefile` - Targets GUI adicionados
- ✅ `scripts/bundle-latest-gui.sh` - Script de bundle para agents GUI
- ✅ `scripts/agent-installers/install-windows.bat` - Instalador Windows
- ✅ `scripts/agent-installers/uninstall-windows.bat` - Desinstalador Windows
- ✅ `scripts/agent-installers/install-linux.sh` - Instalador Linux
- ✅ `scripts/agent-installers/uninstall-linux.sh` - Desinstalador Linux
- ✅ `scripts/agent-installers/zid-agent.service` - Template systemd

**Documentação:**
- ✅ `BUILD-AGENT.md` - Guia de build e dependências
- ✅ `INSTALL-AGENT.md` - Guia de instalação para usuários finais
- ✅ `CLAUDE.md` - Atualizado com nova arquitetura
- ✅ `TODO-AGENTUI.md` - Plano atualizado

### Testes:
- ✅ 11/11 testes unitários passando (`internal/agentui`)
- ⚠️  Build requer dependências do sistema (CGO) - documentado em BUILD-AGENT.md

---

## 1. Visão Geral da Melhoria

**Objetivo:** Transformar o `zid-agent` de um daemon CLI para uma aplicação com interface gráfica que roda na system tray, proporcionando melhor experiência ao usuário final.

**Principais mudanças:**
- Interface gráfica nativa (Windows/Linux)
- Ícone na system tray permanente
- Menu de contexto com opções: "Logs" e "Sair"
- Heartbeat fixo a cada 30 segundos
- Janela de logs para visualização do histórico
- Auto-start opcional no boot do sistema

---

## 2. Biblioteca Recomendada

**getlantern/systray** + **fyne.io** (para janela de logs)

### Por que essas bibliotecas?

**systray:**
- Multiplataforma (Windows, Linux, macOS)
- API simples e estável
- CGO-free para ícone básico
- Suporta menus de contexto nativos
- Projeto maduro e bem mantido

**Fyne:**
- Toolkit GUI puro Go (sem CGO complexo)
- Multiplataforma consistente
- Moderna e fácil de usar
- Widgets prontos (listas, scroll, etc.)
- Boa documentação

**Alternativas consideradas:**
- `go-astilectron`: muito pesado (Electron)
- `lxn/walk`: Windows-only
- `gotk3`: requer GTK3 (CGO complexo)
- `fyne` sozinho: não tem system tray nativo

---

## 3. Arquitetura da Solução

### Estrutura de componentes:

```
┌─────────────────────────────────────┐
│     zid-agent (process)             │
├─────────────────────────────────────┤
│  ┌─────────────────────────────┐   │
│  │   System Tray Manager       │   │ ← systray
│  │   - Ícone                    │   │
│  │   - Menu (Logs, Sair)       │   │
│  └─────────────────────────────┘   │
│                                      │
│  ┌─────────────────────────────┐   │
│  │   Heartbeat Loop (30s)      │   │ ← goroutine
│  │   - Descoberta pfSense       │   │
│  │   - POST heartbeat          │   │
│  └─────────────────────────────┘   │
│                                      │
│  ┌─────────────────────────────┐   │
│  │   Log Manager               │   │ ← circular buffer
│  │   - Ring buffer (500 msgs)  │   │
│  │   - Timestamps              │   │
│  └─────────────────────────────┘   │
│                                      │
│  ┌─────────────────────────────┐   │
│  │   Logs Window (Fyne)        │   │ ← on-demand
│  │   - Lista de mensagens      │   │
│  │   - Auto-scroll             │   │
│  │   - Filtro (futuro)         │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

---

## 4. Estrutura de Arquivos e Pacotes

### Novo layout de diretórios:

```
cmd/zid-agent/
├── main.go                    # Entry point, coordena tudo
├── tray.go                    # Gerenciamento do system tray
├── heartbeat.go               # Loop de heartbeat (isolado)
├── logs.go                    # Ring buffer de logs
├── logsui.go                  # Janela Fyne de logs
└── assets/
    ├── icon.ico               # Ícone Windows
    ├── icon.png               # Ícone Linux
    └── icon_disabled.png      # Ícone quando offline (futuro)

internal/
└── agentui/                   # (novo pacote)
    ├── logger.go              # Logger estruturado thread-safe
    └── ringbuffer.go          # Circular buffer genérico
```

---

## 5. Detalhamento dos Componentes

### 5.1. main.go (Entry Point)

**Responsabilidades:**
- Inicializar configurações (hardcoded: 30s interval, porta 18443)
- Criar log manager (ring buffer de 500 mensagens)
- Iniciar goroutine de heartbeat
- Inicializar system tray (blocking call)

**Fluxo:**
```go
func main() {
    // 1. Setup log manager (thread-safe circular buffer)
    logMgr := NewLogManager(500)

    // 2. Start heartbeat goroutine
    ctx, cancel := context.WithCancel(context.Background())
    go runHeartbeat(ctx, logMgr)

    // 3. Run systray (blocking)
    systray.Run(onReady(logMgr, cancel), onExit)
}
```

---

### 5.2. tray.go (System Tray)

**Responsabilidades:**
- Configurar ícone na system tray
- Criar menu de contexto
- Responder a cliques no menu

**Menu estrutura:**
```
┌─────────────────────────┐
│ ● ZID Agent v1.x        │ (título, desabilitado)
├─────────────────────────┤
│ 📄 Logs                 │ → Abre janela de logs
│ ❌ Sair                 │ → Encerra aplicação
└─────────────────────────┘
```

**Código exemplo:**
```go
func onReady(logMgr *LogManager, cancel context.CancelFunc) func() {
    return func() {
        systray.SetIcon(getIcon())
        systray.SetTitle("ZID Agent")
        systray.SetTooltip("ZID Agent - Running")

        mLogs := systray.AddMenuItem("Logs", "View logs")
        systray.AddSeparator()
        mQuit := systray.AddMenuItem("Quit", "Exit application")

        go func() {
            for {
                select {
                case <-mLogs.ClickedCh:
                    showLogsWindow(logMgr)
                case <-mQuit.ClickedCh:
                    cancel()
                    systray.Quit()
                }
            }
        }()
    }
}
```

---

### 5.3. heartbeat.go (Loop de Heartbeat)

**Responsabilidades:**
- Descobrir pfSense (gateway default → DNS fallback)
- Enviar POST a cada 30 segundos (fixo)
- Logar sucessos/falhas no log manager
- Parar quando contexto for cancelado

**Características:**
- **Intervalo fixo:** 30 segundos (não configurável via flag)
- **Timeout HTTP:** 5 segundos
- **Retry:** tenta gateway primeiro, depois DNS
- **Graceful shutdown:** respeita context cancellation

**Código exemplo:**
```go
func runHeartbeat(ctx context.Context, logMgr *LogManager) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    // First heartbeat immediately
    sendHeartbeat(logMgr)

    for {
        select {
        case <-ticker.C:
            sendHeartbeat(logMgr)
        case <-ctx.Done():
            logMgr.Add("Heartbeat stopped")
            return
        }
    }
}

func sendHeartbeat(logMgr *LogManager) {
    // Descoberta + POST (código atual adaptado)
    // Adiciona logs via logMgr.Add("...")
}
```

---

### 5.4. logs.go (Log Manager com Ring Buffer)

**Responsabilidades:**
- Armazenar últimas N mensagens (circular buffer)
- Thread-safe (mutex)
- Notificar listeners quando há nova mensagem (channel)

**Interface:**
```go
type LogManager struct {
    mu       sync.Mutex
    buffer   []LogEntry
    capacity int
    head     int
    listeners []chan struct{}
}

type LogEntry struct {
    Timestamp time.Time
    Message   string
}

func (lm *LogManager) Add(msg string)
func (lm *LogManager) GetAll() []LogEntry
func (lm *LogManager) Subscribe() <-chan struct{}
```

**Capacidade:** 500 mensagens (configurable)

---

### 5.5. logsui.go (Janela de Logs - Fyne)

**Responsabilidades:**
- Exibir lista de logs com timestamps
- Auto-scroll quando nova mensagem chega
- Permitir scroll manual (desativa auto-scroll temporariamente)
- Botão "Clear" para limpar logs (futuro)

**Layout:**
```
┌──────────────────────────────────────────┐
│  ZID Agent - Logs                    [X] │
├──────────────────────────────────────────┤
│  2025-12-23 10:15:32 | Heartbeat OK      │
│  2025-12-23 10:16:02 | Heartbeat OK      │
│  2025-12-23 10:16:32 | Heartbeat failed  │
│  2025-12-23 10:17:02 | Heartbeat OK      │
│  ...                                      │
│  [auto-scroll zone]                       │
├──────────────────────────────────────────┤
│           [ Clear ]         [ Close ]     │
└──────────────────────────────────────────┘
```

**Características:**
- Janela singleton (só 1 instância por vez)
- Atualização em tempo real via goroutine listening no LogManager
- Formato: `YYYY-MM-DD HH:MM:SS | Message`

**Código exemplo:**
```go
var logsWindow fyne.Window
var logsWindowMutex sync.Mutex

func showLogsWindow(logMgr *LogManager) {
    logsWindowMutex.Lock()
    defer logsWindowMutex.Unlock()

    if logsWindow != nil {
        logsWindow.Show()
        return
    }

    app := app.New()
    logsWindow = app.NewWindow("ZID Agent - Logs")

    list := widget.NewList(
        func() int { return len(logMgr.GetAll()) },
        func() fyne.CanvasObject { return widget.NewLabel("") },
        func(id widget.ListItemID, obj fyne.CanvasObject) {
            entries := logMgr.GetAll()
            label := obj.(*widget.Label)
            label.SetText(entries[id].Format())
        },
    )

    // Auto-update goroutine
    go func() {
        ch := logMgr.Subscribe()
        for range ch {
            list.Refresh()
        }
    }()

    logsWindow.SetContent(list)
    logsWindow.Resize(fyne.NewSize(600, 400))
    logsWindow.Show()

    logsWindow.SetOnClosed(func() {
        logsWindowMutex.Lock()
        logsWindow = nil
        logsWindowMutex.Unlock()
    })
}
```

---

## 6. Funcionalidades do Menu

### Fase 1 (MVP):
- **"Logs"**: Abre janela de logs
- **"Sair"**: Encerra aplicação

### Fase 2 (Futuro):
- **Status indicator**: Ícone muda cor (verde=OK, vermelho=offline)
- **"Pause"**: Pausa heartbeats temporariamente
- **"Settings"**: Configurar porta/DNS (salva em arquivo)
- **"About"**: Versão, build time, licença

---

## 7. Sistema de Logs

### Tipos de mensagens:

| Tipo            | Exemplo                                      |
|-----------------|----------------------------------------------|
| Startup         | `Agent started (version X.Y.Z)`              |
| Heartbeat OK    | `Heartbeat sent successfully to 192.168.1.1` |
| Heartbeat fail  | `Heartbeat failed: connection refused`       |
| Discovery       | `pfSense discovered via gateway: 10.0.0.1`   |
| Shutdown        | `Agent stopped by user`                      |

### Formato:
```
YYYY-MM-DD HH:MM:SS | <Message>
```

### Retenção:
- Últimas **500 mensagens** em memória
- Opcional (futuro): persist to file (`~/.zid-agent/logs.txt`)

---

## 8. Build e Empacotamento

### 8.1. Dependências Go

Adicionar ao `go.mod`:
```bash
go get github.com/getlantern/systray
go get fyne.io/fyne/v2
```

### 8.2. Makefile Updates

```makefile
# Novos targets
build-agent-linux-gui:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) \
	  -o $(BUILD_DIR)/$(AGENT_BINARY)-linux-gui ./cmd/zid-agent

build-agent-windows-gui:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) \
	  -ldflags="-H windowsgui" \
	  -o $(BUILD_DIR)/$(AGENT_BINARY)-windows-gui.exe ./cmd/zid-agent
```

**Nota Windows:** `-ldflags="-H windowsgui"` esconde a janela de console.

### 8.3. Asset Embedding (Ícones)

Usar `embed` package do Go:

```go
//go:embed assets/icon.png
var iconPNG []byte

func getIcon() []byte {
    return iconPNG
}
```

**Ícone recomendado:**
- PNG 256x256 transparente
- Design simples (letra "Z" ou logo da Soul)
- Cores que contrastem com fundos claros/escuros

### 8.4. Bundles Atualizados

Criar novos tarballs:
- `zid-agent-linux-gui-latest.tar.gz` (binário + README)
- `zid-agent-windows-gui-latest.tar.gz` (exe + ícone)

Scripts de instalação:
- **Linux:** systemd user service ou XDG autostart
- **Windows:** atalho na pasta Startup do usuário

---

## 9. Instalação e Auto-Start

### 9.1. Windows

**Instalador básico (batch script):**
```batch
@echo off
echo Installing ZID Agent...
copy zid-agent-windows-gui.exe "%PROGRAMFILES%\ZIDAgent\"
copy install-autostart.vbs "%TEMP%\"
cscript //nologo "%TEMP%\install-autostart.vbs"
echo Done! Agent will start on next login.
```

**Auto-start:**
- Criar atalho em `%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup`
- Ou registro: `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`

### 9.2. Linux

**Systemd user service:** `~/.config/systemd/user/zid-agent.service`
```ini
[Unit]
Description=ZID Agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/zid-agent-linux-gui
Restart=always
RestartSec=30

[Install]
WantedBy=default.target
```

**Enable:**
```bash
systemctl --user enable zid-agent
systemctl --user start zid-agent
```

**Alternativa:** XDG autostart (`~/.config/autostart/zid-agent.desktop`)

---

## 10. Plano de Implementação (Passos)

### **Fase 1: Base da GUI (MVP)**

1. **Setup inicial** ✅ CONCLUÍDO
   - [x] Adicionar dependências: `systray` + `fyne`
   - [x] Criar estrutura de pastas: `cmd/zid-agent/assets/`
   - [x] Adicionar ícones básicos (PNG/ICO)

2. **Log Manager (internal/agentui)** ✅ CONCLUÍDO
   - [x] Implementar `ringbuffer.go` (circular buffer genérico)
   - [x] Implementar `logger.go` (wrapper thread-safe com subscribers)
   - [x] Testes unitários (11/11 testes passando)

3. **Heartbeat isolado (cmd/zid-agent/heartbeat.go)** ✅ CONCLUÍDO
   - [x] Extrair lógica atual de `main.go`
   - [x] Adaptar para usar `LogManager` ao invés de `log.Printf`
   - [x] Fixar intervalo em 30s (removido flag)

4. **System Tray básico (cmd/zid-agent/tray.go)** ✅ CONCLUÍDO
   - [x] Inicializar systray com ícone
   - [x] Menu: "Sair" (funcional)
   - [x] Menu: "Logs" (funcional)

5. **Janela de Logs (cmd/zid-agent/logsui.go)** ✅ CONCLUÍDO
   - [x] UI Fyne com lista de logs
   - [x] Integração com LogManager (read-only)
   - [x] Auto-refresh quando nova mensagem
   - [x] Botões Clear e Close

6. **Integração final (cmd/zid-agent/main.go)** ✅ CONCLUÍDO
   - [x] Coordenar todos os componentes
   - [x] Graceful shutdown
   - [x] Versão e build info

### **Fase 2: Build e Distribuição** ✅ CONCLUÍDO

7. **Build system** ✅ CONCLUÍDO
   - [x] Atualizar Makefile com targets GUI
   - [x] Adicionar target `build-agent-linux-gui`
   - [x] Adicionar target `build-agent-windows-gui`
   - [x] Adicionar target `build-agent-fyne-cross` (alternativa)

8. **Empacotamento** ✅ CONCLUÍDO
   - [x] Scripts de instalação Windows (`install-windows.bat`, `uninstall-windows.bat`)
   - [x] Scripts de instalação Linux (`install-linux.sh`, `uninstall-linux.sh`)
   - [x] Systemd service template (`zid-agent.service`)
   - [x] XDG autostart suportado no instalador Linux
   - [x] Criar `bundle-latest-gui.sh` para bundles GUI
   - [x] README.txt incluído em cada bundle

9. **Documentação** ✅ CONCLUÍDO
   - [x] `INSTALL-AGENT.md` - Guia completo de instalação
   - [x] `BUILD-AGENT.md` - Guia de build e dependências
   - [x] READMEs em bundles (Windows e Linux)

### **Fase 3: Polimento (Pós-MVP)**

10. **Melhorias UX**
    - [x] Ícone com status (verde/vermelho)
    - [x] Tooltip com última conexão
    - [x] Menu "About"

11. **Persistência**
    - [x] Salvar logs em arquivo (~/.zid-agent/logs.txt)
    - [x] Rotação de logs (max 1MB)

12. **Configuração**
    - [x] Menu "Settings" (porta, DNS, interval)
    - [x] Salvar em `~/.zid-agent/config.json`

---

## 11. Riscos e Mitigações

| Risco                          | Probabilidade | Impacto | Mitigação                              |
|-------------------------------|---------------|---------|----------------------------------------|
| CGO issues no cross-compile    | Média         | Alto    | Usar bibliotecas CGO-free (systray ok) |
| Systray não funciona no Wayland| Média         | Médio   | Fallback: XDG tray protocol            |
| Fyne muito pesado              | Baixa         | Médio   | Janela on-demand, não sempre aberta    |
| Usuário fecha janela, pensa que encerrou | Alta | Baixo | Tooltip explica que está na tray      |

---

## 12. Estimativa de Esforço

| Fase                    | Complexidade | Tempo estimado |
|------------------------|--------------|----------------|
| Log Manager            | Baixa        | 2-3 horas      |
| Heartbeat refactor     | Baixa        | 1-2 horas      |
| System Tray            | Média        | 3-4 horas      |
| Janela Logs (Fyne)     | Média        | 4-5 horas      |
| Build/empacotamento    | Média        | 3-4 horas      |
| Testes e debug         | Alta         | 4-6 horas      |
| **Total MVP**          | -            | **~20 horas**  |

---

## 13. Checklist de Entrega

### Código:
- [x] `internal/agentui/` com logger e ringbuffer ✅
- [x] `cmd/zid-agent/` refatorado (tray, heartbeat, logs, logsui) ✅
- [x] Ícones em `cmd/zid-agent/assets/` ✅
- [x] Testes unitários para log manager ✅

### Build:
- [ ] `make build-agent-linux-gui` funcional
- [ ] `make build-agent-windows-gui` funcional
- [ ] Binários testados em Windows 10/11
- [ ] Binários testados em Linux (Ubuntu 22.04+)

### Distribuição:
- [ ] `zid-agent-linux-gui-latest.tar.gz`
- [ ] `zid-agent-windows-gui-latest.tar.gz`
- [ ] Scripts de instalação incluídos
- [ ] README atualizado com instruções

### Documentação:
- [ ] `CLAUDE.md` atualizado com nova arquitetura do agent
- [ ] `CHANGELOG.md` com versão e mudanças
- [ ] Comentários no código (GoDoc style)

---

## 14. Após Implementação

### Versioning:
- Bump para `1.1.0.0` (feature major: GUI)
- Atualizar `Makefile VERSION`
- Atualizar `zid-agent-*-latest.version` files

### Testes:
```bash
# Linux
./build/zid-agent-linux-gui  # Deve aparecer ícone na tray

# Windows (cross-compile test)
GOOS=windows GOARCH=amd64 go build -o test.exe ./cmd/zid-agent
# Testar em VM Windows
```

### Documentação no repositório:
Adicionar ao `CLAUDE.md` na seção de Agents:

```markdown
### 3) Desktop agent: `zid-agent` (Go, Windows/Linux com GUI)
Responsável por:
- Rodar em background com ícone na system tray
- Descobrir o pfSense (gateway → DNS fallback)
- Enviar POST a cada 30s com hostname/username
- Exibir logs em janela Fyne on-demand
- Menu: Logs, Sair (futuro: Settings, About)
```

---

## 15. Estado Atual do Sistema

### Como funciona atualmente:

**Active IPs (no pfSense):**
- O tracker em `internal/activeips/tracker.go` monitora IPs baseado em **tráfego de rede** (ConnStart/ConnEnd/AddBytes)
- Mantém estatísticas: bytes in/out, conexões ativas, timestamps
- O método `SetIdentity()` enriquece IPs **já rastreados** com machine/username vindos do agent
- Possui **Identity TTL**: se o agent não enviar heartbeat dentro do prazo, machine/user são limpos
- Gera snapshot JSON periódico em `/var/run/zid-proxy.active_ips.json`

**Agents (Desktop):**
- Binário CLI simples em `cmd/zid-agent/main.go`
- Descobre pfSense via gateway default ou DNS fallback
- Envia heartbeat JSON a cada 30s (configurável) para `http://<pfsense>:18443/api/v1/agent/heartbeat`
- Payload: `hostname`, `username`, `agent_version`
- Roda em foreground, logs no stdout

**Servidor no pfSense:**
- Recebe heartbeats via `internal/agenthttp/server.go`
- Atualiza Registry (agent) e notifica Active IPs tracker
- Registry tem seu próprio TTL independente

---

## Resumo Executivo

**O que muda:**
- Agent vira aplicação GUI com ícone na tray (Windows/Linux)
- Heartbeat fixo 30s (não configurável via flag)
- Logs visualizáveis em janela dedicada
- Instalação com auto-start opcional

**Bibliotecas:**
- `getlantern/systray` (tray icon + menu)
- `fyne.io/fyne/v2` (janela de logs)

**Complexidade:**
- Média-baixa (bibliotecas maduras, Go puro)
- ~20 horas para MVP completo

**Próximos passos:**
1. Aprovação do plano
2. Implementação fase 1 (MVP)
3. Testes em ambos OS
4. Release e documentação
