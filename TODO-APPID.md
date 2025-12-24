# Plano: Implementação da Aba AppID com Deep Packet Inspection

## Resumo

Implementar um novo binário `zid-appid` que usa **nDPI** (Deep Packet Inspection) para identificar aplicações (Netflix, YouTube, Facebook, etc.) e integrá-lo ao zid-proxy existente, com uma nova aba "AppID" na GUI do pfSense.

## Arquitetura Proposta

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              pfSense                                     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   Internet ◄──► [WAN] ◄──────────────────────────────────► [LAN] ◄──► Clientes
│                           │                                              │
│                           ▼                                              │
│              ┌────────────────────────┐                                  │
│              │      zid-appid         │  ← Bridge inline (como Snort)    │
│              │   (nDPI DPI engine)    │                                  │
│              │   Detecta: netflix,    │                                  │
│              │   youtube, facebook... │                                  │
│              └───────────┬────────────┘                                  │
│                          │ Unix Socket                                   │
│                          │ /var/run/zid-appid.sock                       │
│                          ▼                                               │
│              ┌────────────────────────┐     ┌────────────────────────┐   │
│              │      zid-proxy         │     │      GUI (PHP)         │   │
│              │   (SNI transparent     │◄───►│   - Settings           │   │
│              │    proxy, porta 443)   │     │   - Groups             │   │
│              │   Aplica regras:       │     │   - AppID (NOVA)       │   │
│              │   SNI + AppID + Groups │     │   - Logs unificados    │   │
│              └───────────┬────────────┘     └────────────────────────┘   │
│                          │                                               │
│                          ▼                                               │
│              /var/log/zid-proxy.log                                      │
│              (formato: TS|IP|HOST|GROUP|ACTION|MACHINE|USER|APP)         │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Decisões Técnicas (Confirmadas pelo Usuário)

### Biblioteca DPI: nDPI (via CGO)
- **450+ protocolos** detectados nativamente (Netflix, YouTube, Facebook, WhatsApp, etc.)
- Suporta **Encrypted Traffic Analysis (ETA)** - detecta apps mesmo em HTTPS
- Wrapper Go disponível: [go-ndpi](https://github.com/fs714/go-ndpi)
- Licença: LGPLv3 (compatível)

### Modo de Operação: Bridge Inline ✓
- `zid-appid` opera em modo **bridge inline** (como Snort)
- Tráfego passa através do daemon para análise em tempo real
- Vantagem: Detecção mais precisa dos primeiros pacotes
- Desvantagem: Pode adicionar latência (mitigado pelo nDPI otimizado)

### Comunicação: Unix Socket ✓
- zid-proxy e zid-appid comunicam via **Unix socket** (`/var/run/zid-appid.sock`)
- Mais rápido que HTTP, menor overhead
- Protocolo simples: request/response JSON sobre socket

### Comportamento Padrão: ALLOW ✓
- Se nDPI não detectar app (tráfego muito encriptado): **permite o tráfego**
- Regras SNI existentes continuam sendo aplicadas normalmente
- Apenas regras AppID explícitas (BLOCK_APP) bloqueiam

### Integração com Grupos Existentes
- Reutiliza os grupos já criados em `zid-proxy_groups.php`
- Regras AppID: `BLOCK_APP;grupo;app_name` (ex: `BLOCK_APP;acesso_restrito;netflix`)

### Protocolo Unix Socket

**Socket:** `/var/run/zid-appid.sock`

**Comandos:**
```
# Lookup de app por flow (5-tuple)
→ LOOKUP 192.168.1.100 52.94.228.167 TCP 54321 443
← OK netflix 0.95

# Lookup simplificado por IP (retorna última app detectada)
→ LOOKUP_IP 192.168.1.100
← OK netflix

# App não detectada
→ LOOKUP_IP 192.168.1.50
← UNKNOWN

# Estatísticas
→ STATS
← {"flows_total": 15420, "apps_detected": {"netflix": 234, "youtube": 567, ...}}

# Lista de apps suportadas
→ APPS
← ["netflix", "youtube", "facebook", "whatsapp", ...]
```

**Formato de resposta:**
- `OK <app_name> [confidence]` - App detectada
- `UNKNOWN` - Não detectada (fallback: ALLOW)
- `ERROR <message>` - Erro de comunicação

---

## Componentes a Implementar

### 1. Novo Binário: `cmd/zid-appid/` (Go + CGO/nDPI)

**Arquivos:**
- `cmd/zid-appid/main.go` - Entry point
- `cmd/zid-appid/capture.go` - Captura de pacotes (libpcap)
- `cmd/zid-appid/detector.go` - Wrapper nDPI para detecção
- `cmd/zid-appid/api.go` - API HTTP para comunicação

**Funcionalidades:**
- Opera em modo **bridge inline** entre interfaces
- Usa nDPI para identificar aplicação de cada fluxo em tempo real
- Mantém cache de `flow_key → app_detected` (5-tuple hash)
- Expõe **Unix socket** (`/var/run/zid-appid.sock`):
  - `LOOKUP srcIP dstIP proto srcPort dstPort` → retorna app detectada
  - `STATS` → estatísticas de detecção
- Grava log de detecções (integrado ao log principal)

### 2. Novo Pacote: `internal/appid/`

**Arquivos:**
- `internal/appid/ndpi.go` - Wrapper nDPI (CGO)
- `internal/appid/flow.go` - Estrutura de fluxo e cache
- `internal/appid/rules.go` - Parser de regras AppID
- `internal/appid/matcher.go` - Matching de regras por grupo

**Estruturas:**
```go
type AppRule struct {
    Type      RuleType   // ALLOW_APP ou BLOCK_APP
    GroupName string     // Nome do grupo (ex: "acesso_restrito")
    AppName   string     // Nome da app (ex: "netflix")
}

type FlowInfo struct {
    SrcIP     net.IP
    DstIP     net.IP
    Protocol  string     // "netflix", "youtube", etc.
    FirstSeen time.Time
    LastSeen  time.Time
    Bytes     uint64
}
```

### 3. Modificações no `zid-proxy`

**Arquivos a modificar:**
- `internal/proxy/handler.go` - Consultar AppID antes de aplicar regras
- `internal/rules/rules.go` - Adicionar suporte a regras ALLOW_APP/BLOCK_APP
- `internal/logger/logger.go` - Adicionar coluna APP ao log

**Novo formato de log:**
```
TIMESTAMP | SOURCE_IP | HOSTNAME | GROUP | ACTION | MACHINE | USER | APP
```

Exemplo:
```
2024-12-24T15:30:45Z | 192.168.1.100 | nflxvideo.net | sales | BLOCK | LAPTOP-01 | john | netflix
```

### 4. Nova Aba GUI: `zid-proxy_appid.php`

**Funcionalidades:**
- Lista de aplicações detectáveis (categorizado: Streaming, Social, etc.)
- Checkbox para ALLOW/BLOCK cada app por grupo
- Botão para atualizar lista de apps do nDPI
- Status do daemon `zid-appid` (running/stopped)

**Arquivo de regras:**
- `/usr/local/etc/zid-proxy/appid_rules.txt`

**Formato:**
```
# ZID Proxy AppID Rules
# Format: TYPE;GROUP;APP_NAME
BLOCK_APP;acesso_restrito;netflix
BLOCK_APP;acesso_restrito;youtube
ALLOW_APP;acesso_restrito;microsoft_teams
BLOCK_APP;visitantes;*  # bloqueia todos apps para visitantes
```

### 5. Integração na Aba de Logs

**Modificar:** `pkg-zid-proxy/files/usr/local/www/zid-proxy_log.php`

- Adicionar coluna "App" na tabela de logs
- Filtro por tipo: "Todos", "Proxy (SNI)", "AppID"
- Badge colorido quando APP detectado

### 6. Scripts de Serviço

**Novos arquivos:**
- `scripts/rc.d/zid-appid` - rc.d script para FreeBSD
- `pkg-zid-proxy/files/usr/local/pkg/zid-appid.inc` - Funções PHP

---

## Arquivos a Criar/Modificar

### Novos Arquivos

| Caminho | Descrição |
|---------|-----------|
| `cmd/zid-appid/main.go` | Entry point do daemon DPI |
| `cmd/zid-appid/capture.go` | Captura de pacotes libpcap |
| `cmd/zid-appid/detector.go` | Integração nDPI |
| `cmd/zid-appid/api.go` | API HTTP interna |
| `internal/appid/ndpi.go` | Wrapper CGO para nDPI |
| `internal/appid/flow.go` | Cache de fluxos detectados |
| `internal/appid/rules.go` | Parser de regras AppID |
| `internal/appid/matcher.go` | Matching de grupo+app |
| `internal/appid/client.go` | Cliente Unix socket (usado pelo zid-proxy) |
| `internal/appid/bridge.go` | Implementação do bridge inline |
| `pkg-zid-proxy/files/usr/local/www/zid-proxy_appid.php` | GUI da aba AppID |
| `pkg-zid-proxy/files/usr/local/pkg/zid-appid.inc` | Funções PHP |
| `scripts/rc.d/zid-appid` | Script rc.d |
| `configs/appid_rules.txt` | Template de regras |

### Arquivos a Modificar

| Caminho | Modificação |
|---------|-------------|
| `internal/proxy/handler.go` | Consultar AppID via Unix socket antes de aplicar regras |
| `internal/proxy/server.go` | Adicionar referência ao cliente AppID |
| `internal/rules/rules.go` | Suporte ALLOW_APP/BLOCK_APP para regras por grupo+app |
| `internal/logger/logger.go` | Adicionar coluna APP ao formato de log |
| `cmd/zid-proxy/main.go` | Flag `--appid-socket` para path do socket |
| `pkg-zid-proxy/files/usr/local/pkg/zid-proxy.xml` | Nova aba no menu |
| `pkg-zid-proxy/files/usr/local/www/zid-proxy_log.php` | Coluna APP + filtros |
| `pkg-zid-proxy/files/usr/local/www/zid-proxy_settings.php` | Config do AppID |
| `pkg-zid-proxy/install.sh` | Instalar zid-appid |
| `Makefile` | Target build-appid-freebsd |

---

## Fluxo de Decisão (Runtime)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        FLUXO DE CONEXÃO                                 │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  1. Cliente inicia conexão HTTPS                                        │
│     ↓                                                                   │
│  2. Pacotes passam pelo zid-appid (bridge inline)                       │
│     → nDPI analisa primeiros pacotes                                    │
│     → Detecta app (ex: "netflix")                                       │
│     → Armazena em cache: flow_key → "netflix"                           │
│     ↓                                                                   │
│  3. zid-proxy recebe conexão e extrai SNI                               │
│     ↓                                                                   │
│  4. zid-proxy consulta zid-appid via Unix socket:                       │
│     → LOOKUP 192.168.1.100 52.94.228.167 TCP 54321 443                  │
│     ← netflix                                                           │
│     ↓                                                                   │
│  5. zid-proxy aplica regras (ordem de prioridade):                      │
│     a) Verifica grupo do srcIP                                          │
│     b) Verifica regras AppID (ALLOW_APP/BLOCK_APP) para grupo+app       │
│     c) Se BLOCK_APP match: RST e log com APP=netflix                    │
│     d) Se não match AppID: aplica regras SNI normais                    │
│     e) Default (sem regra): ALLOW                                       │
│     ↓                                                                   │
│  6. Log unificado:                                                      │
│     2024-12-24T15:30:45Z | 192.168.1.100 | nflxvideo.net | sales |      │
│     BLOCK | LAPTOP-01 | john | netflix                                  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Dependências de Build

### Para FreeBSD (pfSense):
```bash
# nDPI library
pkg install ndpi

# Ou compilar do source
git clone https://github.com/ntop/nDPI.git
cd nDPI && ./autogen.sh && ./configure && make && make install
```

### Makefile targets:
```makefile
build-appid-freebsd:
    CGO_ENABLED=1 GOOS=freebsd GOARCH=amd64 \
    go build -o build/zid-appid ./cmd/zid-appid
```

---

## Etapas de Implementação

### Fase 1: Core DPI Engine
- [ ] 1. Criar `internal/appid/` com wrapper nDPI básico
- [ ] 2. Criar `cmd/zid-appid/` com captura e detecção
- [ ] 3. Testar detecção local (Linux) antes de portar para FreeBSD

### Fase 2: Integração com zid-proxy
- [ ] 4. Adicionar coluna APP ao logger
- [ ] 5. Modificar handler.go para consultar AppID
- [ ] 6. Criar formato de regras ALLOW_APP/BLOCK_APP

### Fase 3: GUI pfSense
- [ ] 7. Criar `zid-proxy_appid.php` com lista de apps
- [ ] 8. Modificar `zid-proxy_log.php` para mostrar coluna APP
- [ ] 9. Adicionar tab no `zid-proxy.xml`

### Fase 4: Empacotamento
- [ ] 10. Criar rc.d script para zid-appid
- [ ] 11. Modificar install.sh/update.sh
- [ ] 12. Testar em pfSense real

---

## Considerações

### Performance
- nDPI é otimizado para alta velocidade (10+ Gbps)
- Cache de fluxos reduz overhead de lookup
- zid-appid roda separado do proxy (não bloqueia conexões)

### Limitações
- DPI requer acesso aos primeiros pacotes de cada fluxo
- Alguns apps podem não ser detectados se usarem técnicas de evasão
- Tráfego já estabelecido não será reclassificado

### Alternativa Simplificada (Fallback)
Se DPI for muito complexo, podemos usar **mapeamento SNI→App** como fallback:
- Manter lista de hostnames conhecidos por app (ex: `nflxvideo.net` → Netflix)
- Menos preciso, mas mais simples de implementar
- Pode ser combinado com DPI para melhor cobertura

---

## Apps Suportadas (nDPI)

O nDPI detecta 450+ protocolos. Abaixo os mais relevantes para controle corporativo:

### Streaming Media
| App | Protocolo nDPI |
|-----|----------------|
| Netflix | `netflix`, `netflix_stream` |
| YouTube | `youtube`, `youtube_upload` |
| Spotify | `spotify` |
| Twitch | `twitchtv` |
| Disney+ | Detectado via TLS fingerprint |
| Amazon Prime | `amazon_video` |

### Social Networking
| App | Protocolo nDPI |
|-----|----------------|
| Facebook | `facebook`, `facebook_apps` |
| Instagram | `instagram` |
| Twitter/X | `twitter` |
| TikTok | `tiktok` |
| LinkedIn | `linkedin` |
| WhatsApp | `whatsapp` |

### Messaging
| App | Protocolo nDPI |
|-----|----------------|
| WhatsApp | `whatsapp` |
| Telegram | `telegram` |
| Discord | `discord` |
| Slack | `slack` |
| Microsoft Teams | `ms_teams` |
| Zoom | `zoom` |

### Games
| App | Protocolo nDPI |
|-----|----------------|
| Steam | `steam` |
| PlayStation Network | `playstation` |
| Xbox Live | `xbox` |
| Epic Games | `epicgames` |

### VPN/Tunneling
| App | Protocolo nDPI |
|-----|----------------|
| OpenVPN | `openvpn` |
| WireGuard | `wireguard` |
| Tor | `tor` |
| Proxy genérico | `socks`, `http_proxy` |

---

## Referências Técnicas

- [nDPI - ntop](https://www.ntop.org/products/deep-packet-inspection/ndpi/) - Biblioteca DPI
- [go-ndpi](https://github.com/fs714/go-ndpi) - Wrapper Go para nDPI
- [nDPI Protocols List](https://www.ntop.org/guides/nDPI/protocols.html) - Lista completa de protocolos
- [Application Detection on pfSense](https://www.netgate.com/blog/application-detection-on-pfsense-software) - OpenAppID no pfSense
