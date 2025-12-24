# Plano: Correção do System Tray do ZID-Agent

## Problema Identificado

O system tray do zid-agent não funciona (ícone não aparece, menus não respondem) em Windows e Linux devido a **problemas arquiteturais críticos**:

### Causa Raiz

1. **Conflito de bibliotecas systray**: O projeto usa `github.com/getlantern/systray v1.2.2` diretamente, mas o Fyne também traz `fyne.io/systray` como dependência transitiva. São duas implementações diferentes que conflitam.

2. **Modelo de threading incorreto**:
   - `systray.Run()` bloqueia a main thread
   - `app.New()` do Fyne é chamado via `sync.Once` dentro de um handler de clique do menu (thread errada)
   - Fyne requer que `app.New()` e `app.Run()` sejam chamados na main goroutine

3. **Biblioteca desatualizada**: getlantern/systray v1.2.2 é de 2020 e tem bugs conhecidos no D-Bus (Linux) e problemas de formato de ícone (Windows)

4. **Ícone muito pequeno**: 16x16 pixels é insuficiente para displays modernos

---

## Solução

**Migrar para o suporte nativo de system tray do Fyne** usando a interface `desktop.App` disponível no Fyne 2.4+:
- `SetSystemTrayMenu(menu *fyne.Menu)` - define o menu do tray
- `SetSystemTrayIcon(icon fyne.Resource)` - define o ícone do tray

Isso elimina o conflito de bibliotecas e garante threading correto.

---

## Arquivos a Modificar

| Arquivo | Ação | Descrição |
|---------|------|-----------|
| `go.mod` | Modificar | Remover `github.com/getlantern/systray` |
| `cmd/zid-agent/main.go` | Reescrever | Usar Fyne `app.New()` e `desktop.App` |
| `cmd/zid-agent/tray.go` | Reescrever | Usar `SetSystemTrayMenu`/`SetSystemTrayIcon` |
| `cmd/zid-agent/logsui.go` | Modificar | Receber `fyneApp` como parâmetro, remover `sync.Once` |
| `cmd/zid-agent/assets/icon.go` | Atualizar | Ícone 64x64 PNG |
| `cmd/zid-agent/heartbeat.go` | Sem mudanças | Já está correto |

---

## Implementação Detalhada

### Passo 1: Atualizar go.mod

Remover a dependência direta do getlantern/systray:

```bash
go mod edit -droprequire github.com/getlantern/systray
go mod tidy
```

### Passo 2: Reescrever main.go

```go
package main

import (
    "context"
    "flag"
    "fmt"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/driver/desktop"
    "github.com/guilherme/zid-proxy/internal/agentui"
)

var (
    Version   = "dev"
    BuildTime = "unknown"
)

func main() {
    showVersion := flag.Bool("version", false, "Show version and exit")
    flag.Parse()

    if *showVersion {
        fmt.Printf("zid-agent version %s (built %s)\n", Version, BuildTime)
        return
    }

    // Criar log manager
    logMgr := agentui.NewLogManager(500)
    logMgr.Addf("ZID Agent v%s starting...", Version)

    // Criar context para shutdown gracioso
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Criar Fyne app na main thread
    fyneApp := app.New()

    // Configurar system tray se suportado
    if deskApp, ok := fyneApp.(desktop.App); ok {
        setupSystemTray(deskApp, fyneApp, logMgr, cancel, Version)
    } else {
        logMgr.Add("Warning: System tray not supported")
    }

    // Iniciar heartbeat em goroutine
    go runHeartbeat(ctx, logMgr, Version)

    // Executar Fyne event loop (bloqueia até quit)
    fyneApp.Run()
}
```

### Passo 3: Reescrever tray.go

```go
package main

import (
    "context"
    "fmt"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/driver/desktop"
    "github.com/guilherme/zid-proxy/internal/agentui"
)

// setupSystemTray configura o system tray usando Fyne nativo
func setupSystemTray(deskApp desktop.App, fyneApp fyne.App, logMgr *agentui.LogManager, cancel context.CancelFunc, version string) {
    // Criar itens de menu
    showLogsItem := fyne.NewMenuItem("Logs", func() {
        logMgr.Add("Opening logs window...")
        showLogsWindow(fyneApp, logMgr)
    })

    quitItem := fyne.NewMenuItem("Quit", func() {
        logMgr.Add("Shutting down by user request...")
        cancel()
        fyneApp.Quit()
    })

    // Criar menu
    menu := fyne.NewMenu(
        fmt.Sprintf("ZID Agent v%s", version),
        showLogsItem,
        fyne.NewMenuItemSeparator(),
        quitItem,
    )

    // Configurar menu e ícone do tray
    deskApp.SetSystemTrayMenu(menu)
    icon := fyne.NewStaticResource("icon.png", iconData)
    deskApp.SetSystemTrayIcon(icon)

    logMgr.Add("System tray initialized")
}
```

### Passo 4: Modificar logsui.go

Remover `sync.Once` e `logsApp` global. Receber `fyneApp` como parâmetro:

```go
package main

import (
    "sync"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/widget"
    "github.com/guilherme/zid-proxy/internal/agentui"
)

var (
    logsWindow      fyne.Window
    logsWindowMutex sync.Mutex
)

// showLogsWindow cria ou mostra a janela de logs
func showLogsWindow(fyneApp fyne.App, logMgr *agentui.LogManager) {
    logsWindowMutex.Lock()
    defer logsWindowMutex.Unlock()

    // Se janela já existe, apenas mostrar
    if logsWindow != nil {
        logsWindow.Show()
        logsWindow.RequestFocus()
        return
    }

    // Criar nova janela usando o app existente
    logsWindow = fyneApp.NewWindow("ZID Agent - Logs")

    // ... resto da implementação igual ...

    // Usar SetCloseIntercept para esconder ao invés de fechar
    logsWindow.SetCloseIntercept(func() {
        logsWindow.Hide()
    })

    logsWindow.Show()
}
```

### Passo 5: Criar ícone 64x64

Gerar um ícone PNG 64x64 com a letra "Z" e atualizar `cmd/zid-agent/assets/icon.go` ou usar `//go:embed`.

---

## Ordem de Execução

1. [ ] Criar ícone 64x64 PNG
2. [ ] Atualizar `cmd/zid-agent/assets/icon.go` com novo ícone
3. [ ] Reescrever `cmd/zid-agent/main.go`
4. [ ] Reescrever `cmd/zid-agent/tray.go`
5. [ ] Modificar `cmd/zid-agent/logsui.go`
6. [ ] Atualizar `go.mod` (remover getlantern/systray)
7. [ ] Executar `go mod tidy`
8. [ ] Compilar e testar Linux
9. [ ] Compilar e testar Windows
10. [ ] Atualizar versão no Makefile
11. [ ] Atualizar CHANGELOG.md
12. [ ] Gerar novos bundles

---

## Testes Necessários

1. **Linux**: Verificar que ícone aparece na system tray
2. **Linux**: Clicar "Logs" abre janela sem travar
3. **Linux**: Clicar "Quit" encerra aplicação
4. **Linux**: Heartbeat continua funcionando
5. **Windows**: Repetir todos os testes acima
6. **Ambos**: Fechar janela de logs e reabrir funciona

---

## Riscos e Mitigações

| Risco | Mitigação |
|-------|-----------|
| Fyne systray não disponível em alguns DEs Linux | Adicionar log de warning se `desktop.App` assertion falhar |
| Ícone não visível em temas escuros | Usar ícone com alto contraste |
| `go mod tidy` remove dependências necessárias | Revisar mudanças antes de commitar |

---

## Atualização de Versão

- **Makefile**: `VERSION=1.0.11.4`
- **CHANGELOG.md**: Documentar migração para Fyne systray nativo
