#!/usr/bin/env bash
# Bloqueia, para o celular, a resolução dos hosts do Firebase Authentication
# (Sprint 9: provedor de autenticação indisponível). Sobe um dnsmasq que
# responde NXDOMAIN para os hosts usados pelo SDK Android do Firebase Auth e
# encaminha o resto a resolvedores públicos:
#   - www.googleapis.com (identitytoolkit/v3: entrar com e-mail e senha);
#   - identitytoolkit.googleapis.com (API v1, usada pelos SDKs mais novos);
#   - securetoken.googleapis.com (renovação do ID token).
# O gateway (*.gateway.dev) continua resolvendo normalmente.
#
# No celular, nas opções da rede Wi-Fi, troque o IP para "Estático" e aponte o
# DNS 1 para o IP deste computador (impresso abaixo). Desfaça ao terminar.
#
#   ferramentas/injecao-falhas/dns-bloqueio.sh            # exige sudo (porta 53)
#   ferramentas/injecao-falhas/dns-bloqueio.sh 5353       # porta alternativa, sem sudo (só para testes locais)
set -euo pipefail

PORTA="${1:-53}"
IP="${IP_LOCAL:-$(ip -4 route get 1.1.1.1 | awk '{print $7; exit}')}"
BLOQUEADOS=(www.googleapis.com identitytoolkit.googleapis.com securetoken.googleapis.com)

ARGS=(--no-daemon --port="$PORTA" --listen-address="$IP" --listen-address=127.0.0.1 --bind-interfaces
      --no-resolv --no-hosts --cache-size=0 --server=8.8.8.8 --server=1.1.1.1
      --log-queries --log-facility=-)
for h in "${BLOQUEADOS[@]}"; do ARGS+=(--address="/$h/"); done

echo "dnsmasq em $IP:$PORTA; bloqueados: ${BLOQUEADOS[*]}" >&2
echo "no celular: Wi-Fi > rede atual > IP estático > DNS 1 = $IP" >&2
if [ "$PORTA" = "53" ]; then
  exec sudo dnsmasq "${ARGS[@]}"
else
  exec dnsmasq "${ARGS[@]}"
fi
