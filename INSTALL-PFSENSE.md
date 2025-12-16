# Instalação Rápida - ZID Proxy no pfSense

## 📦 Versão: 1.0.3

Este guia mostra como instalar o zid-proxy-pfsense-v1.0.3.tar.gz no pfSense.

### 🆕 Novidades da v1.0.3

- ⭐ **Menu aparece automaticamente** - Sem necessidade de registro manual
- 🔄 **GUI recarrega automaticamente** - Menu visível imediatamente após instalação
- 📚 **Documentação completa de limitações** - Instruções claras para configurar NAT bypass
- 🛠️ **Processo de instalação 100% automático** - Sem prompts interativos

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
scp zid-proxy-pfsense-v1.0.3.tar.gz root@SEU-PFSENSE-IP:/tmp/
```

### Passo 2: Extrair e instalar

```bash
# Conectar ao pfSense
ssh root@SEU-PFSENSE-IP

# Extrair o pacote
cd /tmp
tar -xzf zid-proxy-pfsense-v1.0.3.tar.gz
cd zid-proxy-pfsense

# Executar instalador
cd pkg-zid-proxy
sh install.sh
```

O instalador irá:
1. ✓ Copiar todos os arquivos necessários
2. ✓ Criar o script RC automaticamente
3. ✓ Perguntar se deseja registrar no pfSense (responda "yes")
4. ✓ Mostrar instruções para completar a instalação

### Passo 3: Verificar instalação

```bash
# Testar o serviço
/usr/local/etc/rc.d/zid-proxy.sh start
/usr/local/etc/rc.d/zid-proxy.sh status

# Ver logs
tail -f /var/log/zid-proxy.log
```

### Passo 4: Acessar interface web

1. Recarregue a interface web do pfSense:
   ```bash
   /usr/local/sbin/pfSsh.php playback reloadwebgui
   ```

2. Acesse: **Services > ZID Proxy**

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
/usr/local/sbin/pfSsh.php playback reloadwebgui
```

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

**Versão do Binário**: 1.0.3
**Data de Build**: 2025-12-16
**Compatível com**: pfSense 2.7.0+ (FreeBSD 15.x)
**SHA256**: `3bba83f8758d0cc5ada06cfcac6410f7be155d4fa42d4b783db60aecdacdeb4e`
