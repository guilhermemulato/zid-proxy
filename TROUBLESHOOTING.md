# Troubleshooting - ZID Proxy

Este documento lista problemas comuns e suas soluções.

## 🔍 Problema: Facebook e outros sites grandes retornam ERR_QUIC_PROTOCOL_ERROR

**Sintoma**:
- Sites como Facebook, Google, etc retornam erro `ERR_QUIC_PROTOCOL_ERROR`
- Ou páginas não carregam corretamente

**Causa**:
Navegadores modernos (Chrome, Edge) tentam usar **QUIC** (HTTP/3 sobre UDP porta 443). O zid-proxy só intercepta **TCP** porta 443, então:
1. Navegador tenta UDP 443 (QUIC)
2. UDP passa direto pelo NAT (não é interceptado)
3. Servidor responde com QUIC
4. Navegador fica confuso porque esperava que a conexão fosse interceptada

**Solução**: Bloquear UDP porta 443 de saída

### Via GUI do pfSense:

1. Vá em **Firewall > Rules > LAN**
2. Clique em **Add** (adicionar regra no topo)
3. Configure:
   - **Action**: Block
   - **Protocol**: UDP
   - **Source**: LAN net
   - **Destination**: any
   - **Destination Port Range**: From 443, To 443
   - **Description**: Block QUIC (HTTP/3) to force TCP for zid-proxy
4. Clique em **Save** e **Apply Changes**

### Via Linha de Comando:

```bash
# Adicionar regra de firewall (temporário - não persiste após reboot)
pfctl -t block_quic -T add 0.0.0.0/0

# Para tornar permanente, use a GUI
```

---

## 🔍 Problema: Regras BLOCK não funcionam

**Sintoma**:
- Criou regra BLOCK via GUI
- Site continua acessível
- Logs mostram ALLOW em vez de BLOCK

**Causa**:
Regras não foram recarregadas automaticamente após salvar via GUI.

**Solução**: Recarregar regras manualmente

```bash
# Método 1: Via script RC
/usr/local/etc/rc.d/zid-proxy.sh reload

# Método 2: Via SIGHUP direto
kill -HUP $(cat /var/run/zid-proxy.pid)

# Método 3: Reiniciar serviço
/usr/local/etc/rc.d/zid-proxy.sh restart
```

**Verificar**:
```bash
# Ver logs para confirmar que regras foram aplicadas
tail -20 /var/log/zid-proxy.log
```

**Correção permanente**: Atualizar para v1.0.2+ onde este bug foi corrigido.

---

## 🔍 Problema: Acesso por IP direto retorna Connection Reset

**Sintoma**:
- Ao acessar `https://192.168.1.1` (pfSense GUI via IP)
- Ou qualquer outro IP privado via HTTPS (https://172.25.200.53)
- Recebe erro "connection reset", "connection closed" ou página não carrega

**Causa**:
Conexões HTTPS para IPs (sem hostname) não enviam **SNI** (Server Name Indication). O proxy não consegue determinar o destino original após NAT redirect. Esta é uma **limitação arquitetônica** do FreeBSD/pf - recuperar o destino original após NAT Port Forward requer divert sockets, que exigiria reescrita completa do proxy.

**Solução Recomendada**: Excluir IPs específicos do NAT redirect

### Via GUI do pfSense:

1. Vá em **Firewall > NAT > Port Forward**
2. Clique para **editar** a regra que redireciona porta 443 para o proxy
3. Na seção **Destination**, configure:
   - ☑ **Invert match** (NOT)
   - **Type**: Single host or alias
   - **Address**: `192.168.1.1` (IP do pfSense)
4. Para múltiplos IPs, crie um **Alias** em:
   - Firewall > Aliases > IP > Add
   - Nome: `bypass_proxy`
   - Type: Host(s)
   - IPs: `192.168.1.1`, `172.25.200.53`, etc.
   - Depois use este alias na regra NAT
5. Clique em **Save** e **Apply Changes**

**Resultado:** Conexões para os IPs especificados NÃO serão redirecionadas ao proxy, permitindo acesso direto.

### Via Linha de Comando (Temporário):

```bash
# Adicionar IPs a uma tabela pf (não persiste após reboot)
pfctl -t bypass_proxy -T add 192.168.1.1
pfctl -t bypass_proxy -T add 172.25.200.53

# Para persistir, use a GUI conforme acima
```

**Alternativas**:

**Opção 2: Usar hostname em vez de IP**

Adicione entrada no DNS local ou arquivo /etc/hosts:
```
192.168.1.1     pfsense.local
172.25.200.53   pfsense.local
```

Acesse via: `https://pfsense.local`

**Opção 3: Mudar porta da GUI do pfSense**

1. **System > Advanced > Admin Access**
2. **TCP Port**: `8443` (ou outra porta diferente de 443)
3. **Save**
4. Acesse via: `https://192.168.1.1:8443`

---

## 🔍 Problema: Não consigo acessar pfSense via https://192.168.1.1 (Legado v1.0.0-v1.0.1)

**Sintoma**:
- Ao tentar acessar GUI do pfSense via IP (https://192.168.1.1)
- Recebe erro `ERR_CONNECTION_RESET` ou conexão recusada

**Causa**:
Conexões HTTPS para IPs (sem hostname) não enviam **SNI** (Server Name Indication). O proxy bloqueia conexões sem SNI por segurança.

**Solução 1**: Excluir pfSense do NAT redirect (Recomendado)

Modificar a regra NAT para NÃO redirecionar tráfego com destino ao IP do pfSense:

1. **Firewall > NAT > Port Forward**
2. Editar a regra que redireciona porta 443
3. Em **Destination**, mudar de "any" para:
   - **Destination**: Invert match (✓)
   - **Type**: Single host or alias
   - **Address**: 192.168.1.1 (IP do pfSense)
4. Salvar

Isso faz com que conexões para 192.168.1.1:443 NÃO sejam redirecionadas ao proxy.

**Solução 2**: Usar hostname em vez de IP

Adicione entrada no DNS local ou arquivo hosts:
```
192.168.1.1  pfsense.local
```

Acesse via: `https://pfsense.local`

**Solução 3**: Desabilitar proxy temporariamente

Para acessar GUI, temporariamente desabilite o NAT redirect em **Firewall > NAT**.

---

## 🔍 Problema: Serviço não inicia

**Sintoma**:
```
service zid-proxy start
# Retorna erro ou serviço não inicia
```

**Diagnóstico**:

```bash
# Ver logs de erro
tail -50 /var/log/system.log | grep zid

# Testar binário diretamente
/usr/local/sbin/zid-proxy -listen :3129 -rules /usr/local/etc/zid-proxy/access_rules.txt -log /tmp/test.log

# Verificar permissões
ls -lh /usr/local/sbin/zid-proxy
ls -lh /usr/local/etc/zid-proxy/access_rules.txt
```

**Soluções comuns**:

1. **Arquivo de regras inválido**:
```bash
# Verificar sintaxe
cat /usr/local/etc/zid-proxy/access_rules.txt

# Recriar arquivo se necessário
rm /usr/local/etc/zid-proxy/access_rules.txt
/usr/local/etc/rc.d/zid-proxy.sh start
```

2. **Porta já em uso**:
```bash
# Verificar se outra coisa está usando porta 3129
sockstat -l | grep 3129
```

3. **Binário corrompido**:
```bash
# Recopiar binário
scp build/zid-proxy root@pfsense:/usr/local/sbin/
chmod +x /usr/local/sbin/zid-proxy
```

---

## 🔍 Problema: Navegação muito lenta

**Sintoma**:
- Sites carregam, mas demorando muito
- Timeouts ocasionais

**Causas possíveis**:

1. **DNS lento** - Proxy precisa resolver hostnames
2. **Regras de firewall** bloqueando conexões de saída do proxy
3. **Muitas regras** - matching lento

**Soluções**:

```bash
# 1. Verificar resolução DNS
time host www.google.com
# Deve responder em menos de 100ms

# 2. Verificar conectividade de saída
telnet www.google.com 443

# 3. Ver quantas regras estão configuradas
wc -l /usr/local/etc/zid-proxy/access_rules.txt

# 4. Logs podem revelar problemas
tail -100 /var/log/zid-proxy.log
```

---

## 🔍 Problema: Logs não são gerados

**Sintoma**:
- Arquivo `/var/log/zid-proxy.log` está vazio ou não existe

**Solução**:

```bash
# Verificar configuração
grep zid_proxy_log /etc/rc.conf.local

# Deve mostrar:
# zid_proxy_log="/var/log/zid-proxy.log"

# Se mostrar /dev/null, corrigir:
vi /etc/rc.conf.local
# Mudar para: zid_proxy_log="/var/log/zid-proxy.log"

# Criar arquivo se não existir
touch /var/log/zid-proxy.log
chmod 644 /var/log/zid-proxy.log

# Reiniciar serviço
/usr/local/etc/rc.d/zid-proxy.sh restart
```

Ou via GUI:
- **Services > ZID Proxy**
- Marcar **☑ Enable Logging**
- Save

---

## 🔍 Problema: Wildcard não funciona nas regras

**Sintoma**:
- Regra `*.example.com` não bloqueia `www.example.com`

**Formato correto**:

```
BLOCK;192.168.1.0/24;*.example.com
```

**Match**:
- ✓ `www.example.com`
- ✓ `api.example.com`
- ✓ `example.com`
- ✗ `notexample.com`

**Teste**:
```bash
# Ver logs para verificar hostname exato
tail -50 /var/log/zid-proxy.log | grep example

# Verificar se regra está no arquivo
cat /usr/local/etc/zid-proxy/access_rules.txt
```

---

## 🔍 Problema: Menu "Services > ZID Proxy" não aparece

**Sintoma**:
- Pacote instalado mas menu não aparece na GUI

**Solução**:

```bash
# Registrar pacote
cd /tmp/zid-proxy-pfsense/pkg-zid-proxy
php register-package.php

# Recarregar interface web
/usr/local/sbin/pfSsh.php playback reloadwebgui

# Ou reiniciar pfSense
shutdown -r now
```

---

## 📊 Comandos Úteis para Diagnóstico

```bash
# Ver status do serviço
/usr/local/etc/rc.d/zid-proxy.sh status

# Ver processo rodando
ps aux | grep zid-proxy

# Ver porta em escuta
sockstat -l | grep 3129

# Ver últimos logs
tail -50 /var/log/zid-proxy.log

# Ver logs em tempo real
tail -f /var/log/zid-proxy.log

# Testar regra específica
grep "example.com" /usr/local/etc/zid-proxy/access_rules.txt

# Ver conexões ativas do proxy
pfctl -ss | grep 3129

# Script de diagnóstico completo
cd /tmp/zid-proxy-pfsense/pkg-zid-proxy
sh diagnose.sh
```

---

## 🆘 Ainda com Problemas?

Se nenhuma solução acima funcionou:

1. Execute o diagnóstico completo:
```bash
cd /tmp/zid-proxy-pfsense/pkg-zid-proxy
sh diagnose.sh > /tmp/zid-diagnostic.txt
cat /tmp/zid-diagnostic.txt
```

2. Colete logs:
```bash
tail -100 /var/log/zid-proxy.log > /tmp/zid-proxy.log.txt
tail -100 /var/log/system.log | grep zid > /tmp/zid-system.log.txt
```

3. Abra um issue no GitHub com:
   - Versão do pfSense
   - Versão do zid-proxy
   - Resultado do `diagnose.sh`
   - Logs relevantes
   - Descrição detalhada do problema

---

**Versão**: 1.0.2
**Última atualização**: 2025-12-16
