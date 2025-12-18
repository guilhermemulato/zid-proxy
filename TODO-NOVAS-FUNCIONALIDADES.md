## Novas funcionalidades (requisitos)

### 1) Criacao de um rotate logs
- [ ] Deve ter 1 log por dia
- [ ] A quantidade de dias deve ser definida na Aba Settings da Web Gui do pfsense (Default 7 dias)
- [ ] Formato pode ser igual ao logs do sistema *(system.log, system.log.0, system.log.1)
- [ ] Criar um binario em Go apenas para isso (futuro: enviar logs para webhook externo)
- [ ] Rodar via cron do pfSense (1h em 1h), independente do binario principal

### 2) Aba Settings do Web GUI (pfSense)
- [ ] Adicionar a versao atual do binario do zid-proxy
- [ ] Adicionar um botao de update (mesmo processo: `sh /usr/local/sbin/zid-proxy-update`)
- [ ] Mostrar resultado final na tela: se ja estiver na ultima versao mostrar isso; se atualizou mostrar apenas `done`
- [ ] Adicionar controles do servico: start (se parado), stop/restart (se rodando)

### 3) Servico de watchdog
- [ ] Monitorar se o servico `zid-proxy` esta rodando
- [ ] Se nao estiver, iniciar o servico SOMENTE se `Enable` estiver ON nas configuracoes
- [ ] Criar agendamento na cron do pfSense para monitorar isso
- [ ] Ajustar instalacao para criar a cron caso nao exista
- [ ] Ajustar uninstall para remover a cron

---

## Plano de execucao (por fases)

### Fase 0 — Base de versao e integracoes
- [ ] Confirmar formato do `zid-proxy -version` (compatível com o parse do updater atual)
- [ ] Manter versao coerente entre: `Makefile`, `CHANGELOG.md`, `zid-proxy-pfsense-latest.version`
- [ ] (Opcional) Persistir versao instalada em arquivo (fallback quando binario nao for executavel)

### Fase 1 — Binario Go de logrotate (`zid-proxy-logrotate`)
- [ ] Criar `cmd/zid-proxy-logrotate/` (binario separado)
- [ ] Criar `internal/logrotate/` com logica testavel (sem dependencias do pfSense)
- [ ] Implementar rotacao diaria no padrao: `zid-proxy.log`, `zid-proxy.log.0`, `zid-proxy.log.1`, ...
- [ ] Respeitar `keepDays` (default 7)
- [ ] Criar flags: `-log`, `-keep-days` (e opcional: `-pidfile` / `-hup` se for sinalizar o daemon)
- [ ] Adicionar testes unitarios (`*_test.go`) cobrindo os principais cenarios

### Fase 2 — Reabrir log no binario principal (para rotacao funcionar)
- [ ] Implementar `Reopen()` no logger (reabrir o arquivo de log com lock)
- [ ] No `SIGHUP` do `zid-proxy`: recarregar regras e reabrir o log
- [ ] Ajustar o `zid-proxy-logrotate` para sinalizar `SIGHUP` quando rotacionar (se aplicavel)

### Fase 3 — pfSense Web GUI (Settings)
- [ ] Adicionar campo `log_retention_days` no `zid-proxy.xml` (default 7 + validacao)
- [ ] Incluir defaults/validacao no `zid-proxy.inc`
- [ ] Mostrar versao instalada na aba Settings (via `/usr/local/sbin/zid-proxy -version`)
- [ ] Botao Update: executar `sh /usr/local/sbin/zid-proxy-update` e exibir somente `done` quando atualizar
- [ ] Controles do servico: Start/Stop/Restart (via rc.d / service-utils)

### Fase 4 — Cron do logrotate (1h/1h) usando `install_cron_job()` do `services.inc`
- [ ] Usar sempre a funcao `install_cron_job()` do `services.inc` (nao editar crontab manualmente)
- [ ] Instalar/atualizar cron do logrotate no `install` do pacote (idempotente)
- [ ] Remover cron do logrotate no `uninstall` do pacote
- [ ] Agendamento: minuto `0`, hora `*` (1h em 1h), usuario `root`
- [ ] Comando final do cron chama o `zid-proxy-logrotate` com os parametros corretos

### Fase 5 — Watchdog (cron + logica “so inicia se Enable=on”)
- [ ] Implementar watchdog (script/PHP) que verifica `Enable` e o status do processo
- [ ] Se `Enable=off`: watchdog nao inicia nada
- [ ] Se `Enable=on` e servico parado: iniciar via rc.d
- [ ] Criar cron do watchdog usando `install_cron_job()` (idempotente)
- [ ] No `resync`: ativar/desativar o cron do watchdog conforme `Enable`
- [ ] No `uninstall`: remover o cron do watchdog

### Fase 6 — Release (pacote `latest` para SCP/update)
- [ ] Bump de versao (alteracao pequena: `1.0.9.1`, `1.0.9.2`, etc)
- [ ] Registrar alteracoes no `CHANGELOG.md` criando nova versao
- [ ] Rodar `make test`
- [ ] Gerar binarios: `make build-freebsd`
- [ ] Atualizar `zid-proxy-pfsense-latest.version`
- [ ] Gerar `zid-proxy-pfsense-latest.tar.gz` (bundle com versao `latest`)


