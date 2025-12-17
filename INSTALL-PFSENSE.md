# Instalação Rápida - ZID Proxy no pfSense

## 📦 Versão: 1.0.8

Este guia mostra como instalar o zid-proxy-pfsense-v1.0.8.tar.gz no pfSense.

### 🆕 Novidades da v1.0.8

- 🔥 **CRITICAL BUG FIX** - BLOCK rules agora funcionam corretamente!
- ✅ **Regras BLOCK Respeitadas** - Não mais ignoradas por ALLOW rules anteriores
- 🎯 **Prioridade Correta** - ALLOW > BLOCK funciona como documentado
- 🛠️ **Lógica Corrigida** - Match() verifica TODAS as regras antes de decidir

### Novidades da v1.0.7

- 📊 **Settings Table Display** - Aba Settings agora mostra tabela com configuração atual
- ✅ **Visual Feedback** - Veja status (Enable, Interface, Port, Logging, Timeout) sem clicar em Edit
- 🎨 **UX Melhorado** - Não mais tabela vazia com apenas ícones

### Novidades da v1.0.6

- 🚀 **Log em Tempo Real** - Logs aparecem em ≤1 segundo (não mais 3 minutos de atraso)
- 🔧 **Flush Automático** - Logger faz flush a cada 1 segundo (ticker ativado)
- ⚡ **Performance** - Overhead mínimo, compatível com pfSense 2.8.1/FreeBSD 15

### Novidades da v1.0.5

- 🎯 **GUI Reload Corrigido** - Usa `/etc/rc.restart_webgui` oficial (sem erro 502)
- 📊 **Auto-Refresh Configurável** - Selecione: Disabled, 5s, 10s, 20s, 30s
- ⏸️ **Pause Auto-Refresh** - Checkbox para pausar e analisar logs
- 🔍 **Filtro em Tempo Real** - Busca instantânea por IP ou domínio
- 💾 **Filtro Persistente** - Filtro mantém-se ativo durante auto-refresh

### ⚠️ Limitações Conhecidas

#### 1. Acesso por IP Direto (Sem SNI)

Para acessar o pfSense GUI ou outros serviços por IP (https://192.168.1.1, https://172.25.200.53), você **PRECISA excluir** estes IPs do NAT redirect. Esta é uma limitação arquitetônica - conexões HTTPS para IPs não enviam SNI, e o proxy não consegue determinar o destino original após NAT.

**Solução Obrigatória (escolha uma):**

**Opção 1 - Excluir IPs do NAT** (Recomendado):
1. **Firewall > NAT > Port Forward**
2. Editar a regra que redireciona porta 443
3. **Destination**: Invert match (☑) → Single host or alias → `192.168.1.1`
4. Para múltiplos IPs, crie um Alias em Firewall > Aliases > IP
5. Save & Apply Changes

**Opção 2 - Usar Hostname**:
- Adicionar `192.168.1.1 pfsense.local` no DNS ou /etc/hosts
- Acessar via `https://pfsense.local`

**Opção 3 - Mudar Porta da GUI**:
- System > Advanced > Admin Access > TCP Port: `8443`
- Acessar via `https://192.168.1.1:8443`

#### 2. QUIC/HTTP3 (Facebook, Google)

Sites grandes usam QUIC (HTTP/3 sobre UDP). Para funcionar, você **DEVE bloquear UDP porta 443**.

**Solução:**
1. **Firewall > Rules > LAN > Add**
2. Action: **Block**, Protocol: **UDP**
3. Source: **LAN net**, Destination: **any**, Port: **443**
4. Description: "Block QUIC to force TCP for zid-proxy"
5. Save & Apply Changes

Veja [TROUBLESHOOTING.md](TROUBLESHOOTING.md) para detalhes.

## 🚀 Instalação Rápida

### Passo 1: Copiar arquivo para o pfSense

```bash
# Do seu computador
scp zid-proxy-pfsense-v1.0.8.tar.gz root@SEU-PFSENSE-IP:/tmp/
```

### Passo 2: Extrair e instalar

```bash
# Conectar ao pfSense
ssh root@SEU-PFSENSE-IP

# Extrair o pacote
cd /tmp
tar -xzf zid-proxy-pfsense-v1.0.8.tar.gz
cd zid-proxy-pfsense

# Executar instalador
cd pkg-zid-proxy
sh install.sh
```

O instalador irá:
1. ✓ Copiar todos os arquivos necessários
2. ✓ Criar o script RC automaticamente
3. ✓ Registrar o pacote no pfSense (adiciona tags <package> e <menu>)
4. ✓ Reiniciar PHP-FPM para carregar o menu
5. ✓ Menu "Services > ZID Proxy" aparece automaticamente!

### Passo 3: Verificar instalação

```bash
# Testar o serviço
/usr/local/etc/rc.d/zid-proxy.sh start
/usr/local/etc/rc.d/zid-proxy.sh status

# Ver logs
tail -f /var/log/zid-proxy.log
```

### Passo 4: Acessar interface web

1. Aguarde ~10 segundos para GUI recarregar

2. Recarregue seu navegador (Ctrl+Shift+R)

3. Acesse: **Services > ZID Proxy** (deve aparecer automaticamente!)

3. Configure:
   - ☑ Enable
   - Interface: LAN
   - Port: 3129 (ou conforme sua necessidade)
   - ☑ Enable Logging

4. Adicione regras na aba **Access Rules**

5. Configure NAT redirect em **Firewall > NAT > Port Forward**

---

## 🔧 Solução de Problemas

### Erro: "service not found"

Se o comando `service zid-proxy` não funcionar:

```bash
cd /tmp/zid-proxy-pfsense/pkg-zid-proxy
php activate-package.php
```

### Menu não aparece na GUI

```bash
cd /tmp/zid-proxy-pfsense/pkg-zid-proxy
php register-package.php
/etc/rc.restart_webgui
```

Aguarde ~10 segundos e recarregue o navegador (Ctrl+Shift+R).

### Diagnóstico completo

```bash
cd /tmp/zid-proxy-pfsense/pkg-zid-proxy
sh diagnose.sh
```

### Reinstalar do zero

```bash
cd /tmp/zid-proxy-pfsense/pkg-zid-proxy
sh uninstall.sh
sh install.sh
```

---

## 📋 Estrutura do Pacote

```
zid-proxy-pfsense/
├── build/zid-proxy              # Binário para FreeBSD
├── pkg-zid-proxy/
│   ├── install.sh               # Instalador principal ⭐
│   ├── activate-package.php     # Cria RC script
│   ├── register-package.php     # Registra no pfSense
│   ├── diagnose.sh              # Diagnóstico
│   ├── uninstall.sh             # Desinstalador
│   ├── README.md                # Documentação detalhada
│   └── files/                   # Arquivos do pacote
├── scripts/rc.d/zid-proxy       # Script RC standalone
├── configs/access_rules.txt     # Regras de exemplo
└── README.md                    # Documentação geral
```

---

## 📖 Documentação

- **Documentação completa**: `README.md`
- **Troubleshooting detalhado**: `pkg-zid-proxy/README.md`
- **Instruções para desenvolvedores**: `CLAUDE.md`

---

## ✅ Verificação Pós-Instalação

Execute este checklist:

- [ ] Binário instalado: `ls -lh /usr/local/sbin/zid-proxy`
- [ ] RC script existe: `ls -lh /usr/local/etc/rc.d/zid-proxy.sh`
- [ ] Serviço inicia: `/usr/local/etc/rc.d/zid-proxy.sh start`
- [ ] Processo rodando: `ps aux | grep zid-proxy`
- [ ] Menu aparece na GUI: Services > ZID Proxy
- [ ] Log funciona: `tail /var/log/zid-proxy.log`

---

## 🆘 Suporte

Se encontrar problemas:

1. Execute: `cd /tmp/zid-proxy-pfsense/pkg-zid-proxy && sh diagnose.sh`
2. Verifique os logs: `tail -100 /var/log/zid-proxy.log`
3. Leia: `pkg-zid-proxy/README.md`

---

## 📋 Changelog

### v1.0.8 (2025-12-17)
- 🔥 **CRITICAL BUG FIX**: BLOCK rules now work correctly
- ✅ **Fixed rule matching logic**: No longer returns ALLOW immediately when first ALLOW rule matches
- 🎯 **Priority fixed**: ALLOW > BLOCK now works as documented - checks ALL rules before deciding
- 🛠️ **Core logic corrected**: `Match()` function rewritten to evaluate all matching rules
- ✨ **All tests passing**: Unit tests confirm correct behavior restoration

### v1.0.7 (2025-12-17)
- 📊 **UX Improvement**: Settings tab displays configuration summary table with 5 columns
- ✅ **Visibility**: Shows Enable, Interface, Port, Logging, and Timeout values at a glance
- 🎨 **No More Empty Table**: Replaced icon-only display with informative configuration summary
- 🛠️ **XML Update**: Added `<adddeleteeditpagefields>` section to package definition

### v1.0.6 (2025-12-17)
- 🚀 **Log Latency Fixed**: Reduced from 3 minutes to ≤1 second on pfSense 2.8.1/FreeBSD 15
- 🔧 **Auto Flush**: Activated automatic log buffer flush every 1 second
- ⚡ **Performance**: Minimal overhead (1 flush/second), huge UX improvement
- 📝 **Technical**: Fixed buffered I/O issue where logs remained in 4KB buffer indefinitely

### v1.0.5 (2025-12-17)
- 🎯 **GUI Reload Corrigido**: Usa `/etc/rc.restart_webgui` oficial do pfSense (sem erro 502)
- 📊 **Tela de Log Melhorada**: Auto-refresh configurável (5s, 10s, 20s, 30s, Disabled)
- ⏸️ **Pause Auto-Refresh**: Checkbox para pausar e analisar logs detalhadamente
- 🔍 **Filtro em Tempo Real**: Busca instantânea por IP ou domínio enquanto digita
- 💾 **Filtro Persistente**: Mantém filtro ativo durante auto-refresh e reloads
- 📝 **Backend + Frontend**: Filtro aplicado em PHP (otimização) e JavaScript (UX)

### v1.0.4 (2025-12-17)
- ✅ **Menu 100% funcional**: Tag `<menu>` agora adicionada corretamente ao config.xml
- ✅ **Auto-start funciona**: Serviço inicia automaticamente após reboot do pfSense
- 🔧 **Correção crítica**: register-package.php reescrito para adicionar menu ao config.xml
- 📝 **Convenções corretas**: Usa `configurationfile` em vez de `config_file`
- 🚀 **PHP-FPM correto**: install.sh usa `onerestart` em vez de `reloadwebgui`
- 🎯 **Interface padrão**: Mudado de 'lan' para 'all' para melhor compatibilidade com NAT

### v1.0.3 (2025-12-16)
- ⭐ **Menu automático**: Services > ZID Proxy aparece sem intervenção manual
- 🔄 Install.sh registra pacote e recarrega GUI automaticamente
- 📚 Documentação completa de limitações arquitetônicas
- 🛠️ Instruções detalhadas para configurar NAT bypass para acesso por IP

### v1.0.2 (2025-12-16)
- 🔥 **Critical Fix**: BLOCK rules agora aplicam imediatamente após salvar via GUI
- 🏠 Suporte para acesso a IPs privados sem SNI (https://192.168.1.1 funciona)
- 📚 Adicionado TROUBLESHOOTING.md com soluções para problemas comuns
- 🔧 Reload de regras agora usa restart do serviço (mais confiável que SIGHUP)

### v1.0.1 (2025-12-16)
- ✨ Adicionada opção "All Interfaces" na GUI
- 🔧 Correção: GUI não sobrescreve mais listen address incorretamente
- 📝 Interface padrão agora é "All Interfaces" para melhor compatibilidade com NAT
- ✅ Proxy continua funcionando após salvar configurações via GUI

### v1.0.0 (2025-12-16)
- 🎉 Release inicial
- ✓ Proxy transparente SNI com filtragem IP+hostname
- ✓ Interface web completa para pfSense
- ✓ Scripts de instalação automatizados

---

**Versão do Binário**: 1.0.8
**Data de Build**: 2025-12-17
**Compatível com**: pfSense 2.7.0+ / 2.8.1 (FreeBSD 14.x / 15.x)
**SHA256**: `<calculated after build>`
