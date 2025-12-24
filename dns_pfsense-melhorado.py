import threading
import ipaddress
from linecache import cache
from pyrad.client import Client
from pyrad.dictionary import Dictionary
from pyrad.packet import AccessRequest, AccessAccept, AccessReject
from flask import Flask, render_template, request, jsonify, Response, session, flash, redirect, url_for
from scapy.all import *
from scapy.layers.dns import DNSQR
import subprocess
import flask
import os
import sys
import json
import socket
from cachetools import TTLCache
from scapy.all import get_if_addr
from dns.resolver import resolve, NoAnswer, NXDOMAIN
from dns import resolver, exception
from _datetime import datetime
from functools import wraps
from werkzeug.utils import redirect

list_soc = []
list_threads = []
stop_event = threading.Event()
sniffers = []
control = True


def dns_resolver(pkt, sock):
    sock_out = sock
    global hostname_zid
    sites_drop = []
    pkt = pkt
    pkt.show()

    if pkt.haslayer(DNS) and pkt.haslayer(UDP):
        src_host = pkt[IP].src
        dst_host = pkt[IP].dst
        src_port = pkt[UDP].sport
        dst_port = pkt[UDP].dport
        qname = pkt[DNSQR].qname
        qtype = pkt[DNS].qd.qtype

        print(f'{src_host}:{src_port} -> {dst_host}:{dst_port} = {qname}')
        if src_host in cache_grupos:
            acl_grupo = cache_grupos[src_host]

        else:
            """VERIFICANDO QUAL GRUPO O IP PERTENCE"""
            gr = "default"

            encontrado = False
            for grupo, valor in grupos.items():
                if encontrado is False:
                    for item in valor if isinstance(valor, list) else [valor]:
                        print(f"{grupo} -> {item}")
                        if src_host == item:
                            print(f"IP encontrado em {grupo}")
                            encontrado = True
                            gr = grupo
                            cache_grupos[src_host] = gr
                            break
            if encontrado is False:
                for grupo, valor in grupos.items():
                    for item in valor if isinstance(valor, list) else [valor]:
                        print(f"{grupo} -> {item}")
                        address = ipaddress.ip_address(src_host)
                        net = ipaddress.ip_network(item, strict=False)
                        if address in net:
                            print(f"IP encontrado em {grupo}")
                            encontrado = True
                            gr = grupo
                            cache_grupos[src_host] = gr
                            break
            acl_grupo = gr

        """MONTANDO A LISTA DE SITES BLOQUEADOS E LIBERADOS"""

        sites_drop = sites_bloqueados_por_acl[acl_grupo]
        print(f"Drop: {sites_drop}")
        sites_allow = sites_liberados_por_acl[acl_grupo]
        print(f"Access: {sites_allow}")

        qname = qname.decode()
        hostname = qname[:-1]
        print(hostname)

        # if any(site in hostname for site in sites_drop) and not any(site in hostname for site in sites_allow):
        # if hostname in sites_drop and hostname not in sites_allow:
        # if any(site in hostname.endswith (for site in sites_drop)) and not any(site in hostname.endswith( for site in sites_allow)):
        if hostname.endswith(tuple(sites_drop)) and not hostname.endswith(tuple(sites_allow)):
            print("Site bloqueado")
            bloqueado = True
            add_log(src_host, qname, bloqueado)

        else:
            custom_resolver = resolver.Resolver()
            custom_resolver.timeout = 1.0  # Tempo por tentativa individual (segundos)
            custom_resolver.lifetime = 2.0
            print("Site liberado")
            bloqueado = False

            if qtype == 12:

                # qname_str = qname.decode()
                print(f"Requisição PTR recebida: {qname}")

                if qname.endswith(".in-addr.arpa.") or qname.endswith(".ip6.arpa."):
                    # Retornando nome falso para satisfazer o pfSense
                    dns_reply = (
                            IP(dst=src_host, src=dst_host) /
                            UDP(dport=src_port, sport=dst_port) /
                            DNS(
                                id=pkt[DNS].id,
                                qr=1,
                                aa=1,
                                rd=1,
                                ra=1,
                                qd=pkt[DNS].qd,
                                an=DNSRR(rrname=qname, type="PTR", ttl=300, rdata=hostname_zid),
                                ar=DNSRROPT(rclass=4096)
                            )
                    )
                    sock_out.sendto(bytes(dns_reply[DNS]), (src_host, src_port))
                    print(f"Resposta PTR enviada para {qname}")
                    add_log(src_host, qname, False)
                    return
                else:
                    # PTR não reconhecido
                    dns_reply = (
                            IP(dst=src_host, src=dst_host) /
                            UDP(dport=src_port, sport=dst_port) /
                            DNS(
                                id=pkt[DNS].id,
                                qr=1,
                                aa=1,
                                rd=1,
                                ra=1,
                                qd=pkt[DNS].qd,
                                rcode=3,  # NXDOMAIN
                                ar=DNSRROPT(rclass=4096)
                            )
                    )
                    sock_out.sendto(bytes(dns_reply[DNS]), (src_host, src_port))
                    print("PTR desconhecido - NXDOMAIN enviado")
                    add_log(src_host, qname, True)
                    return

            elif qtype == 28:

                print("IPv6")
                if qname in cache_dnsv6:
                    print(f"Ja esta salvo")
                    print(f"cache v6: {cache_dnsv6}")
                    with lock:
                        hostname = cache_dnsv6[qname]

                    dns_reply = (
                            IP(dst=src_host, src=dst_host) /
                            UDP(dport=src_port, sport=dst_port) /
                            DNS(
                                id=pkt[DNS].id,
                                qr=1,
                                aa=1,
                                rd=1,
                                ra=1,
                                qd=pkt[DNS].qd,
                                an=DNSRR(
                                    rrname=qname,
                                    type="AAAA",
                                    rclass="IN",
                                    ttl=300,
                                    rdata=hostname
                                ),
                                ar=DNSRROPT(rclass=4096)
                            )
                    )

                    sock_out.sendto(bytes(dns_reply[DNS]), (src_host, src_port))
                    add_log(src_host, qname, bloqueado)

                else:

                    hostname = ""
                    try:
                        resposta = custom_resolver.resolve(qname, 'AAAA')
                        hostname = resposta[0].to_text()

                    except (exception.Timeout, exception.DNSException) as e:
                        print("Erro na consulta DNSv6")
                        return

                    if hostname:
                        dns_reply = (
                                IP(dst=src_host, src=dst_host) /
                                UDP(dport=src_port, sport=dst_port) /
                                DNS(
                                    id=pkt[DNS].id,
                                    qr=1,
                                    aa=1,
                                    rd=1,
                                    ra=1,
                                    qd=pkt[DNS].qd,
                                    an=DNSRR(
                                        rrname=qname,
                                        type="AAAA",
                                        rclass="IN",
                                        ttl=300,
                                        rdata=hostname
                                    ),
                                    ar=DNSRROPT(rclass=4096)
                                )
                        )

                        sock_out.sendto(bytes(dns_reply[DNS]), (src_host, src_port))
                        add_log(src_host, qname, bloqueado)
                        add_cachev6(qname, hostname)


                    else:

                        dns_reply = (
                                IP(dst=src_host, src=dst_host) /
                                UDP(dport=src_port, sport=dst_port) /
                                DNS(
                                    id=pkt[DNS].id,
                                    qr=1,  # resposta
                                    aa=1,
                                    rd=1,
                                    ra=1,
                                    qd=pkt[DNS].qd,
                                    rcode=3,  # NXDOMAIN
                                    ar=DNSRROPT(rclass=4096)
                                )
                        )
                        sock_out.sendto(bytes(dns_reply[DNS]), (src_host, src_port))

            elif qtype in [15, 16, 33, 6, 2, 5]:
                print(f"Tipo de requisição DNS não suportada: {qtype}")
                dns_reply = (
                        IP(dst=src_host, src=dst_host) /
                        UDP(dport=src_port, sport=dst_port) /
                        DNS(
                            id=pkt[DNS].id,
                            qr=1,
                            aa=1,
                            rd=1,
                            ra=1,
                            qd=pkt[DNS].qd,
                            rcode=3,  # NXDOMAIN
                            ar=DNSRROPT(rclass=4096)
                        )
                )
                sock_out.sendto(bytes(dns_reply[DNS]), (src_host, src_port))
                add_log(src_host, qname, bloqueado)
                return

            elif qtype == 255:  # ANY
                print("Requisição ANY (255) - não implementada")
                dns_reply = (
                        IP(dst=src_host, src=dst_host) /
                        UDP(dport=src_port, sport=dst_port) /
                        DNS(
                            id=pkt[DNS].id,
                            qr=1,
                            aa=1,
                            rd=1,
                            ra=1,
                            qd=pkt[DNS].qd,
                            rcode=4,  # Not Implemented
                            ar=DNSRROPT(rclass=4096)
                        )
                )
                sock_out.sendto(bytes(dns_reply[DNS]), (src_host, src_port))
                add_log(src_host, qname, bloqueado)
                return

            elif qtype == 1:
                hostname = ""
                qname_limpo = qname.strip('.')
                if qname_limpo in apontamentos:
                    print("Apontamento Manual")
                    with lock:
                        hostname = apontamentos.get(qname_limpo)
                        print(hostname)

                elif qname in cache_dns:
                    print(f"Ja esta salvo")
                    with lock:
                        hostname = cache_dns[qname]

                else:

                    try:
                        hostname = resolve(qname, 'A')[0]
                        # hostname = str.hostname
                        print(f"DNS Resolvido {hostname}")
                    except:
                        print("Erro na consulta DNS")
                        return

                if hostname:

                    dns_reply = (
                            IP(dst=src_host, src=dst_host) /
                            UDP(dport=src_port, sport=dst_port) /
                            DNS(
                                id=pkt[DNS].id,
                                qr=1,
                                aa=1,
                                rd=1,
                                ra=1,
                                qd=pkt[DNS].qd,
                                an=DNSRR(rrname=qname, ttl=300, rdata=hostname),
                                ar=DNSRROPT(rclass=4096)
                            )
                    )

                    sock_out.sendto(bytes(dns_reply[DNS]), (src_host, src_port))
                    print(f"cache: {cache_dns}")
                    add_cache(qname, hostname)
                    add_log(src_host, qname, bloqueado)


                else:
                    print("Erro na consulta DNS")
            else:
                print(f"Condicao final de {src_host}")
                dns_reply = (
                        IP(dst=src_host, src=dst_host) /
                        UDP(dport=src_port, sport=dst_port) /
                        DNS(
                            id=pkt[DNS].id,
                            qr=1,  # resposta
                            aa=1,  # autoridade
                            rd=1,
                            ra=1,
                            qd=pkt[DNS].qd,
                            an=None,
                            ar=DNSRROPT(rclass=4096)
                        )
                )

    else:
        print("Pacote sem camada DNS")


def add_log(src_host, qname, bloqueado):
    data = datetime.now()
    data_format = data.strftime("%Y-%m-%d %H:%M")
    if bloqueado is True:
        with lock:
            msg = f"\n{data_format} {src_host} TCP_DENIED 0 {qname} {src_host}"
            with open(log, 'a') as file:
                file.write(msg)
    else:
        with lock:
            msg = f"\n{data_format} {src_host} TCP_MISS 0 {qname} {src_host}"
            with open(log, 'a') as file:
                file.write(msg)


def add_cache(qname, hostname):
    with lock:
        if qname in cache_dns:
            print("qname ja esta em cache")

        else:
            cache_dns[qname] = hostname


def add_cachev6(qname, hostname):
    with lock:
        if qname in cache_dns:
            print("qname ja esta em cache")

        else:
            cache_dnsv6[qname] = hostname


"""Função que usa o Thread para permitir requisições simultaneas"""


def thread_dns(pkt):
    thread = threading.Thread(target=dns_resolver, args=(pkt,))
    thread.start()


def parar_sniffers():
    global sniffers
    for sniffer in sniffers:
        try:
            sniffer.stop()
        except Exception as e:
            print(f"Erro ao parar sniffer: {e}")
    sniffers.clear()


def carregar_config():
    if os.path.exists(config_file):
        with open(config_file, 'r') as f:
            return json.load(f)

    else:
        return {
            'grupos': {'default': [], 'full': [], 'controlado': []},
            'acl': {'default': ['default'], 'full': ['full'], 'controlado': ['controlado']},
            'acl_sites': {'default': [], 'full': [], 'controlado': []},
            'acl_sites_manual': {'default': [], 'full': [], 'controlado': []},
            'acl_sites_allow': {'default': [], 'full': [], 'controlado': []},
            'start': False,
            'interfaces': []
        }


def carregar_entradas():
    if os.path.exists(manual_entry):
        with open(manual_entry, 'r') as f:
            return json.load(f)

    else:
        dados_entry = {
            "entradas": {

            }
        }

        with open(manual_entry, 'w') as f:
            json.dump(dados_entry, f, indent=4)

        with open(manual_entry, 'r') as f:
            return json.load(f)


def salvar_config(config):
    """Salva arquivo de config json"""
    with lock_save:
        with open(config_file, 'w') as f:
            json.dump(config, f, indent=4)


# INICIO FLASK
dns_filter = Flask(__name__)
dns_filter.secret_key = 'nF%fHY9L56£T4x2Zg5'

dns_filter.config['TEMPLATES_AUTO_RELOAD'] = True

log = rf"/usr/local/dns_filter/log/log.txt"
config_file = rf"/usr/local/dns_filter/dns_config.json"
manual_entry = rf"/usr/local/dns_filter/manual_entry.json"
social_nets_file = rf"/usr/local/dns_filter/lists/social_nets.txt"
gov_file = rf"/usr/local/dns_filter/lists/gov.txt"
porn_file = rf"/usr/local/dns_filter/lists/porn.txt"
doh_file = rf"/usr/local/dns_filter/lists/doh.txt"
gamble_file = rf"/usr/local/dns_filter/lists/gamble.txt"
sports_file = rf"/usr/local/dns_filter/lists/sports.txt"
videogames_file = rf"/usr/local/dns_filter/lists/videogames.txt"
music_streaming_file = rf"/usr/local/dns_filter/lists/music_streaming.txt"
video_streaming_file = rf"/usr/local/dns_filter/lists/video_streaming.txt"
webcommerce_file = rf"/usr/local/dns_filter/lists/webcommerce.txt"

with open(social_nets_file, 'r') as file:
    social_nets = file.read()

with open(gov_file, 'r') as file:
    gov = file.read()

with open(porn_file, 'r') as file:
    porn = file.read()

with open(doh_file, 'r') as file:
    doh = file.read()

with open(gamble_file, 'r') as file:
    gamble = file.read()

with open(sports_file, 'r') as file:
    sports = file.read()

with open(videogames_file, 'r') as file:
    videogames = file.read()

with open(music_streaming_file, 'r') as file:
    music_streaming = file.read()

with open(video_streaming_file, 'r') as file:
    video_streaming = file.read()

with open(webcommerce_file, 'r') as file:
    webcommerce = file.read()

# GRUPOS/CONFIG
categories = ['porn', 'social_nets', 'gov', 'doh', 'gamble', 'sports', 'videogames', 'music_streaming',
              'video_streaming', 'webcommerce']

acl_sites_format = {}

# VARIAVEIS GLOBAIS
grupos = {}
acl = {}
acl_sites = {}
acl_sites_allow = {}
acl_sites_manual = {}
sites_bloqueados_por_acl = {}
sites_liberados_por_acl = {}
start = bool
interfaces = []
cache_grupos = TTLCache(maxsize=300, ttl=300)
apontamentos = {}


def reload_config():
    global grupos
    global acl
    global acl_sites
    global acl_sites_allow
    global acl_sites_manual
    global sites_bloqueados_por_acl
    global sites_liberados_por_acl
    global start
    global interfaces
    global apontamentos
    global hostname_zid
    global list_soc

    list_soc = []

    hostname_zid = socket.getfqdn()

    config = carregar_config()
    grupos = config['grupos']
    acl = config['acl']
    acl_sites = config['acl_sites']
    acl_sites_manual = config['acl_sites_manual']
    start = config['start']
    interfaces = config['interfaces']

    entrada_manual = carregar_entradas()
    apontamentos = entrada_manual['entradas']

    # FAZ MONTANDO A LISTA QUE SEJA ENVIADA PARA O FRONT PARA EXEBIBIR APENAS SITES INFORMADOS MANUALMENTE
    acl_sites_front = acl_sites
    print(f"acl limpa: {acl_sites_front}")
    print(f"ativo: {start}")

    social_nets_list = social_nets.split()
    porn_list = porn.split()
    doh_list = doh.split()
    gov_list = gov.split()
    gamble_list = gamble.split()
    sport_list = sports.split()
    videogames_list = videogames.split()
    music_streaming_list = music_streaming.split()
    video_streaming_list = video_streaming.split()
    webcommerce_list = webcommerce.split()

    """Alterado aqui"""
    for cha, val in acl_sites.items():
        lista_ok = []
        for i in val:
            if i == "social_nets.list":
                i_ok = social_nets_list
                lista_ok.append(i_ok)
            elif i == "porn.list":
                i_ok = porn_list
                lista_ok.append(i_ok)
            elif i == "doh.list":
                i_ok = doh_list
                lista_ok.append(i_ok)
            elif i == "gov.list":
                i_ok = gov_list
                lista_ok.append(i_ok)
            elif i == "gamble.list":
                i_ok = gamble_list
                lista_ok.append(i_ok)

            elif i == "sport.list":
                i_ok = sport_list
                lista_ok.append(i_ok)

            elif i == "videogames.list":
                i_ok = videogames_list
                lista_ok.append(i_ok)

            elif i == "music_streaming.list":
                i_ok = music_streaming_list
                lista_ok.append(i_ok)

            elif i == "video_streaming.list":
                i_ok = video_streaming_list
                lista_ok.append(i_ok)

            elif i == "webcommerce.list":
                i_ok = webcommerce_list
                lista_ok.append(i_ok)

            else:
                lista_ok.append(i)

        print(f"lista iterado: {lista_ok}")
        acl_sites[cha] = lista_ok

    acl_sites_allow = config['acl_sites_allow']

    # acl_sites = acl_sites_format
    print(f"Lista: {acl_sites}")
    print(grupos)
    print(acl)

    """CRIANDO ACL_DROP COM BASE NOS GRUPOS"""
    for grupo, entradas in acl_sites.items():
        lista = []
        for item in entradas:
            if isinstance(item, str):
                lista.extend(item.strip().splitlines())
            elif isinstance(item, list):
                for subitem in item:
                    lista.extend(subitem.strip().splitlines())
        sites_bloqueados_por_acl[grupo] = lista

    """CRIANDO ACL_ALLOW COM BASE NOS GRUPOS"""

    sites_liberados_por_acl = {}
    for grupo, entradas in acl_sites_allow.items():
        lista_allow = []
        for item in entradas:
            if isinstance(item, str):
                lista_allow.extend(item.strip().splitlines())
            elif isinstance(item, list):
                for subitem in item:
                    lista_allow.extend(subitem.strip().splitlines())
        sites_liberados_por_acl[grupo] = lista_allow
    # sites_drop = social_nets.strip().splitlines() + gov.strip().splitlines()
    # sites_drop = []


reload_config()
# print(sites_drop)
# USA TTLCACHE PARA FAZER O CONTROLE DO CACHE DEFININDO O MAXIMO DE ENTRADAS E O TEMPO LIMITE
cache_dns = TTLCache(maxsize=300, ttl=300)
cache_dnsv6 = TTLCache(maxsize=300, ttl=300)
# apontamentos = {}
# apontamentos = {"unifi":"10.255.0.1", "unifi2": "10.255.0.2"}

# USA O LOCK PARA NÃO TER CONCORRENCIA NA HORA DE ADICIONAR OU LER AS ENTRADAS NO DICIONARIO CACHE
lock = threading.Lock()
lock_save = threading.Lock()
print(f"Cache DNS: {cache_dns}")


# Valida se o usuario ja esta logado
def login_required(f):
    @wraps(f)
    def decorated_function(*args, **kwargs):
        if 'logged_in' not in session or not session['logged_in']:
            flash('Por favor, faça login para acessar esta página.', 'warning')
            return redirect(url_for('index'))
        return f(*args, **kwargs)

    return decorated_function


@dns_filter.route('/')
def index():
    return render_template('index.html')


@dns_filter.route('/autenticacao', methods=['POST'])
def autenticacao():
    user = request.form.get("usuario")
    senha = request.form.get("senha")
    liberado = False

    if user == "admin" and senha == "_int@383@SoulZ3#":
        liberado = True
        session['logged_in'] = True
        session['username'] = user
    else:
        # Configurações do servidor RADIUS
        server = "10.234.0.1"
        secret = b"87aea2a2437e1dda8630d163420c8a"
        username = user
        password = senha

        # Dicionário de atributos RADIUS (padrão)
        radius_dict = Dictionary("/usr/local/dns_filter/dictionary")

        # Cria o cliente RADIUS
        client = Client(server=server, secret=secret, dict=radius_dict)
        client.AuthPort = 1812  # Porta padrão RADIUS para autenticação

        # Cria o pacote Access-Request
        req = client.CreateAuthPacket(code=AccessRequest, User_Name=user)
        req["User-Password"] = req.PwCrypt(senha)

        # Envia o pacote e aguarda resposta
        try:
            reply = client.SendPacket(req)

            if reply.code == AccessAccept:
                print("✅ Acesso permitido!")
                liberado = True
                session['logged_in'] = True
                session['username'] = user
            elif reply.code == AccessReject:
                print("Acesso negado!")
            else:
                print(f"Código de resposta desconhecido: {reply.code}")
        except Exception as e:
            print(f"Erro ao se conectar ao servidor RADIUS: {e}")

    if liberado is True:

        for nome, sites in acl_sites.items():
            nova_lista = []
            for item in sites:
                if isinstance(item, list):
                    nova_lista.extend(item)  # Desempacota listas aninhadas
                else:
                    nova_lista.append(item)
            acl_sites[nome] = nova_lista
        print(f"acl sites: {acl_sites}")
        print(f"acl_manual:{acl_sites_manual}")

        return render_template('login_sucess.html', interfaces=interfaces, interfaces_select=interfaces_select,
                               start=start, grupos=grupos, acl=acl, acl_sites=acl_sites,
                               acl_sites_manual=acl_sites_manual, acl_sites_allow=acl_sites_allow, listas=categories)

    if liberado is False:
        return render_template('index.html', mensagem="Credenciais invalidas")


@dns_filter.route('/salvar_grupo', methods=['POST', 'GET'])
@login_required
def salvar_grupo():
    dados = request.get_json()
    grupos_save = dados.get('grupos', {})
    acl_save = dados.get('acl', {})
    acl_sites_save = dados.get('acl_sites', {})
    acl_sites_manual = dados.get('acl_sites_manual', {})
    interfaces_save = dados.get('interfaces')
    start_save = dados.get('start')
    print(f"Enable: {start_save} and List: {interfaces_save}")
    print(f"acl_sites_antes: {acl_sites_save}")

    # LIST COMPRESSION PARA REMOVER ENTRADAS .LIST E DEIXAR APENAS AS ENTRADAS MANUAIS
    """for chave, val in acl_sites_manual.items():
        acl_sites_manual[chave] = [i for i in val if ".list" not in i]"""

    print(f"acl_manual: {acl_sites_manual}")

    print(f"Acl_save: {acl_sites_save}")
    acl_sites_allow_save = dados.get('acl_sites_allow', {})
    dici_config = {
        'grupos': grupos_save,
        'acl': acl_save,
        'acl_sites': acl_sites_save,
        'acl_sites_manual': acl_sites_manual,
        'acl_sites_allow': acl_sites_allow_save,
        'start': start_save,
        'interfaces': interfaces_save
    }

    salvar_config(dici_config)
    reload_config()
    cache_grupos.clear()
    parar_soc()

    """Editei aqui"""
    # carregar_config()
    flash('Ajustes atualizados com sucesso!', 'success')
    return jsonify({"mensagem": "Ajustes atualizados com sucesso!"})


@dns_filter.route('/real_time', methods=['GET'])
@login_required
def real_time():
    return render_template('real_time.html')


@dns_filter.route('/get_real_time', methods=['GET'])
@login_required
def get_real_time():
    comando = ['tail', '-n', '100', '/usr/local/dns_filter/log/log.txt']
    tail = subprocess.run(comando, capture_output=True, text=True)
    registros = tail.stdout
    return Response(registros, mimetype='text/plain')


@dns_filter.route('/logout', methods=['POST'])
def logout():
    session.pop('logged_in', None)
    session.pop('username', None)
    flash('Você foi desconectado.', 'info')
    return render_template('index.html')


# Obtem lista de interfaces
interfaces_select = get_if_list()

"""FAZ UMA ITERAÇÃO PARA PERMITIR ESCUTAR EM MAIS DE UMA INTERFACE CASO NECESSARIO E DIVIDINDO CADA INTERFACE EM UM THREAD SEPARADO"""


def inicio_dns_filter():
    global list_soc, list_threads
    print(f"Apontamentos: {apontamentos}")

    if start is True:
        for porta in interfaces:
            address = get_if_addr(porta)
            print(porta, address)
            subprocess.run(["ifconfig", porta, "promisc"])
            t = threading.Thread(target=sock, args=(address,))
            t.start()
            list_threads.append(t)


    else:
        print("DNS desativado")


def sock(address):
    global list_threads, control, stop_event
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.bind((address, 53))  # Reserva a porta
        sock.settimeout(1.0)
        list_soc.append(sock)  # Armazena para manter vivo
        print(f"[*] Servidor DNS escutando em {address}:53... (apenas para reserva)")
        while not stop_event.is_set():
            try:
                data, addr = sock.recvfrom(512)

                try:
                    # Converte o pacote usando Scapy
                    dns_layer = DNS(data)
                    src_ip = addr[0]
                    src_port = addr[1]
                    pkt = IP(dst=address, src=src_ip) / UDP(dport=53, sport=src_port) / dns_layer

                    t = threading.Thread(target=dns_resolver, args=(pkt, sock))
                    t.start()
                    list_threads.append(t)

                except Exception as e:
                    print(f"[!] Erro ao processar pacote recebido: {e}")

            except socket.timeout:
                continue

            except Exception as e:
                print(f"[!] Erro inesperado no socket {address}: {e}")
                break

        sock.close()
    except:
        print("Socket não aberto")


def parar_soc():
    global list_threads, list_soc, stop_event
    stop_event.set()
    print("[*] Encerrando sockets e threads antigas...")
    time.sleep(1.0)  # Dá tempo para threads pararem
    print(list_threads)
    for s in list_soc:
        s.close()
    for s in list_threads:
        s.join()

    for porta in interfaces:
        subprocess.run(["ifconfig", porta, "-promisc"])

    list_threads = []
    list_soc = []
    stop_event.clear()
    inicio_dns_filter()

    print("[*] Todos os threads e sockets encerrados.")


inicio_dns_filter()
if __name__ == '__main__':
    dns_filter.run(host="0.0.0.0", port=5000,
       ssl_context=("/usr/local/dns_filter/dns_filter.crt", "/usr/local/dns_filter/dns_filter.key"))

