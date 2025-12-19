# AGENTS.md (Guia do Repositório) — zid-proxy (pfSense) + zid-agent (Desktop)

Este arquivo descreve, em português (Brasil), como o projeto funciona, sua arquitetura, onde ficam os arquivos e como fazer build/atualização. A intenção é que qualquer pessoa que chegar agora consiga entender rapidamente “o que é o quê”.

## Visão Geral

O **zid-proxy** é um proxy transparente TCP para HTTPS que faz filtragem baseada em:
- **IP de origem** (cliente) e
- **Hostname** extraído do **SNI** (TLS ClientHello).

Ele roda no **pfSense (FreeBSD)** e integra com uma GUI (páginas PHP) no pfSense para gerenciar regras e visualizar conexões.

Além disso, existe o **zid-agent** (Windows/Linux), que roda no computador do usuário e envia “heartbeats” para o pfSense com:
- `hostname` (nome da máquina) e
- `username` (usuário logado),

para enriquecer as telas de “Active IPs” e “Logs”.

## Componentes do Sistema

### 1) Daemon no pfSense: `zid-proxy` (Go)
Responsável por:
- Escutar conexões TCP (normalmente tráfego redirecionado via NAT/port-forward para porta do proxy).
- Ler TLS ClientHello e extrair **SNI**.
- Aplicar regras (ALLOW/BLOCK) combinando `srcIP + hostname`.
- Logar conexões em arquivo.
- Gerar snapshot JSON dos IPs ativos (tráfego agregado por IP).
- Expor uma API HTTP (somente LAN) para receber heartbeats do agent.

### 2) Integração GUI no pfSense: `pkg-zid-proxy` (PHP/XML/SH)
Responsável por:
- Criar as abas do pacote em **Services > ZID Proxy**.
- Persistir configurações no `config.xml` do pfSense.
- Gerar `rc.conf` e o script `rc.d` que inicia o daemon com os flags corretos.
- Instalar/atualizar/remover arquivos de UI e scripts auxiliares.

### 3) Desktop agent: `zid-agent` (Go, Windows/Linux)
Responsável por:
- Descobrir o pfSense (primeiro tenta **gateway default**, depois fallback **DNS** `zid-proxy.lan`).
- Enviar `POST` periódico para o pfSense com `hostname/username`.

## Fluxos Principais

### Fluxo A — Proxy e regras (SNI)
1. Cliente abre HTTPS (TCP/443) e é redirecionado para o `zid-proxy`.
2. `zid-proxy` lê ClientHello, extrai SNI.
3. Aplica regras:
   - **ALLOW tem prioridade** sobre **BLOCK**
   - Default: **ALLOW** se não houver match
4. Se BLOCK: fecha com RST (linger 0).
5. Se ALLOW: conecta no upstream `hostname:443` e faz proxy bidirecional.

### Fluxo B — Active IPs (tráfego agregado por IP)
1. O tracker registra conexões/bytes por IP (`internal/activeips`).
2. Periodicamente é gerado o JSON: `/var/run/zid-proxy.active_ips.json`.
3. A aba “Active IPs” (`zid-proxy_active_ips.php`) lê esse JSON e exibe.

### Fluxo C — Agent (hostname/user → IP) e TTL
1. `zid-agent` envia heartbeat para o pfSense:
   - `POST http://<gateway>:18443/api/v1/agent/heartbeat`
2. O `zid-proxy` registra a identidade do IP de origem.
3. A identidade tem **TTL (sem heartbeat)**: após X segundos sem heartbeat, `Machine/User` ficam vazios (mesmo que o IP continue ativo por tráfego).

### Fluxo D — Logs enriquecidos na GUI
1. O `zid-proxy` grava logs no arquivo (sempre).
2. A aba “Logs” lê o arquivo e também lê o snapshot de Active IPs:
   - se o `source_ip` do log estiver **ativo**, a UI mostra badges `Machine/User`.
   - se não estiver ativo, mostra vazio.

## Formatos de Arquivo

### Regras (arquivo)
Local: `/usr/local/etc/zid-proxy/access_rules.txt`

Formato (modo legacy):
`TYPE;IP_OR_CIDR;HOSTNAME`

Exemplos:
- `BLOCK;192.168.1.0/24;*.facebook.com`
- `ALLOW;192.168.1.100;*.facebook.com`

### Log (arquivo)
Local: `/var/log/zid-proxy.log`

Formato atual:
`TIMESTAMP | SOURCE_IP | HOSTNAME | GROUP | ACTION | MACHINE | USER`

Obs.: `MACHINE` e `USER` podem estar vazios.

### Active IPs (JSON snapshot)
Local: `/var/run/zid-proxy.active_ips.json`

Contém IPs agregados e, quando disponível, `machine/username`.

## Estrutura do Repositório (o que cada grupo faz)

### Go (daemon e agent)
- `cmd/zid-proxy/` — binário do daemon no pfSense.
- `cmd/zid-agent/` — binário do agent (Windows/Linux).
- `cmd/zid-proxy-logrotate/` — helper para rotação de log.
- `internal/`
  - `activeips/` — tracker e snapshot de IPs ativos (bytes/conns) + identidade com TTL.
  - `agent/` — registry de identidades (IP → machine/user) para uso em runtime/logs.
  - `agenthttp/` — API HTTP do agent (endpoint de heartbeat).
  - `config/` — defaults e configuração usada pelo daemon.
  - `gateway/` — descoberta de gateway default (Linux/Windows) usada pelo agent.
  - `logger/` — logger estruturado (formato com colunas separadas por `|`).
  - `logrotate/` — rotação diária numérica do log.
  - `proxy/` — listener TCP, handler, proxy bidirecional.
  - `rules/` — parser/matcher de regras (legacy e/ou suporte a grupos conforme GUI).
  - `sni/` — parsing de ClientHello para extrair SNI.

### pfSense package (GUI/instalação)
- `pkg-zid-proxy/files/` — “root filesystem” do pacote (o que vai para `/usr/local/...` no pfSense).
  - `pkg-zid-proxy/files/usr/local/pkg/` — include PHP e `zid-proxy.xml` (abas/menu).
  - `pkg-zid-proxy/files/usr/local/www/` — páginas da GUI:
    - `zid-proxy_settings.php` — configurações do daemon e controles de serviço.
    - `zid-proxy_agent.php` — configurações do listener/TTL do agent.
    - `zid-proxy_active_ips.php` — lista de IPs ativos (snapshot JSON).
    - `zid-proxy_log.php` — visualização do log (com enrichment via Active IPs).
    - `zid-proxy_groups.php` — gestão de grupos (modo groups).
    - `zid-proxy_rules.php` — regras legacy.
- `pkg-zid-proxy/install.sh` — instala/atualiza os arquivos no pfSense (não apaga config.xml).
- `pkg-zid-proxy/update.sh` — updater “completo” (baixa bundle, extrai, roda `install.sh`).
- `pkg-zid-proxy/update-bootstrap.sh` — updater “bootstrap” instalado em `/usr/local/sbin/zid-proxy-update`.
- `pkg-zid-proxy/uninstall.sh` — remove arquivos do pacote.
- `pkg-zid-proxy/diagnose.sh` — diagnóstico de instalação/arquivos.
- `pkg-zid-proxy/pkg-plist` — lista de arquivos do pacote (importante manter atualizado).

### Build e artefatos
- `build/` — binários gerados localmente (não versionar).
- `dist/` — staging para empacotar bundles (não versionar).
- `scripts/bundle-latest.sh` — monta os bundles `latest` e atualiza `sha256.txt`.
- `zid-proxy-pfsense-latest.version` — arquivo de versão usado pelo updater para decidir “Already up-to-date”.
- `sha256.txt` — checksums dos bundles.

## Comandos de Build e Testes

```bash
make test

# Binários do pfSense (FreeBSD/amd64)
make build-freebsd

# Agents para testes (Linux/Windows amd64)
make build-agent-linux
make build-agent-windows

# Empacotar bundles latest (gera 3 tarballs + sha256.txt)
make bundle-latest
```

## Bundles (sempre separados)

O processo de release gera 3 arquivos:
- `zid-proxy-pfsense-latest.tar.gz` (pfSense: binários + pkg-zid-proxy + scripts)
- `zid-agent-linux-latest.tar.gz` (agent Linux)
- `zid-agent-windows-latest.tar.gz` (agent Windows)

## Atualização no Cliente (pfSense)

No pfSense, o arquivo `/usr/local/sbin/zid-proxy-update` é o ponto de entrada recomendado:
- Ele verifica a versão remota comparando com `...latest.version`.
- Se houver versão nova, baixa o bundle e roda o `update.sh` embarcado.

Exemplos:
```sh
sh /usr/local/sbin/zid-proxy-update
sh /usr/local/sbin/zid-proxy-update -f
sh /usr/local/sbin/zid-proxy-update -u https://.../zid-proxy-pfsense-latest.tar.gz
```

## Padrões e Regras de Desenvolvimento

- Go: sempre rodar `gofmt -w .` antes de entregar alterações.
- Testes: preferir testes determinísticos em `internal/*/*_test.go`.
- Mudou código? Atualize o `CHANGELOG.md` e **bump de versão** no `Makefile`.
- Alteração pequena: use sufixo incremental (ex.: `1.0.11.3.2.4`).
- Ao final, gere novamente os bundles (`make bundle-latest`) e garanta:
  - `zid-proxy-pfsense-latest.version` atualizado
  - `sha256.txt` atualizado

## Referências Técnicas

- SNI proxy pattern: https://www.agwa.name/blog/post/writing_an_sni_proxy_in_go
- Desenvolvimento de pacotes pfSense: https://docs.netgate.com/pfsense/en/latest/development/develop-packages.html
