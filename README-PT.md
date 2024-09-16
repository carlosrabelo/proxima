# Proxima

Proxima é uma CLI em Go que constrói localmente uma imagem qcow2 Alpine Linux pronta para Docker e a provisiona como VM no Proxmox VE.

## Destaques

- Build local de qcow2 Alpine via Docker e `alpine-make-vm-image`
- Provisionamento Proxmox via SSH `qm` (create, importdisk, attach, boot, agent)
- Inicialização da VM e resolução do IPv4 do convidado via QEMU Guest Agent (`WaitIP`)
- Arquivo de configuração e flags de CLI para host, autenticação, tamanho da VM e storage
- Linhas de status por estágio e limpeza de artefatos parciais em falha

## Pré-requisitos

- **Go 1.23+** — necessário para compilar a partir do código-fonte
- **Docker** — CLI no host para o build local do qcow2
- **SSH** até o nó Proxmox
- **Proxmox VE** com `qm`, storage (padrão `local-lvm`) e bridge (padrão `vmbr0`)
- DHCP (ou outra atribuição de IP) na bridge do convidado para o `WaitIP` observar um endereço

## Instalação

### Compilar a partir do código-fonte

```bash
git clone <repo-url>
cd proxima
make build
make install         # ~/.local/bin/proxima
make uninstall
make package         # .deb em dist/ (binário em /usr/bin)
sudo dpkg -i dist/*.deb
```

Binário após o build: `bin/proxima`.

## Uso

Somente build local do qcow2:

```bash
proxima build
```

Pipeline completo (Validate → BuildLocal → Upload → Provision → Start → WaitIP por padrão):

```bash
proxima create
```

Iniciar uma VM já provisionada e imprimir o IPv4 do convidado:

```bash
proxima start
```

Aguardar / imprimir o IPv4 do convidado sem iniciar:

```bash
proxima ip
```

## Configuração

Copie [`docs/proxima.example.yaml`](docs/proxima.example.yaml) para `vms/Proxima.yaml` (só local; `vms/` fica no git só com `.gitkeep`). Esse manifesto é lido automaticamente quando `-c` é omitido. Detalhes: [`docs/configuration.md`](docs/configuration.md).

Ordem sem `-c`: `./Proxima.yaml`, depois `./vms/Proxima.yaml`. Use `-c path` só para sobrescrever. Coloque o disco gerado em `vms/` (por exemplo `output: "vms/alpine-ready.qcow2"`).

Autenticação (exatamente um modo):

- Senha: `user` / `password` (ou env `PROXIMA_PASSWORD`)
- Token de API: `token_id` / `token_secret` (ou env `PROXIMA_TOKEN_SECRET`); no modo token também é necessário `ssh_private_key_path`

Senha root do convidado: `root_password` ou env `PROXIMA_ROOT_PASSWORD`.

SSH opcional no convidado: `guest_ssh_public_key_path` (ou `--guest-ssh-public-key`) grava um `.pub` OpenSSH em `/root/.ssh/authorized_keys` no build da imagem. Reconstrua o qcow2 e recrie a VM depois de alterar.

Flags de create / start: `--start` / `--wait-ip` (padrão true), `--agent-timeout` (padrão `120s`), `--agent-poll-interval` (padrão `2s`).

Validar (na raiz do repositório):

```bash
proxima config check
```

## Estágios do pipeline

| Estágio | Papel |
| --- | --- |
| Validate | Carregar e validar a configuração |
| BuildLocal | Construir o qcow2 Alpine pronto para Docker localmente |
| Upload | Enviar o disco para `/var/tmp/proxima/<vmid>-disk.qcow2` |
| Provision | `qm create` / `importdisk` / attach scsi0 / boot / agent |
| Start | Iniciar a VM |
| WaitIP | Resolver o IPv4 do convidado via QEMU Guest Agent |

## Códigos de saída

| Código | Significado |
| --- | --- |
| 0 | Sucesso |
| 1 | Erro de uso / validação |
| 2 | Falha de build local / Docker |
| 3 | Falha de transporte ou API Proxmox |
| 4 | Falha na limpeza após provisionamento parcial |

## Desenvolvimento

```bash
make build           # Compila o binário em bin/proxima
make test            # Roda todos os testes
make quality         # Checagem de formato, vet, test e build
make install         # Instala o binário em ~/.local/bin
make uninstall       # Remove de ~/.local/bin
make package         # Gera .deb em dist/ (binário em /usr/bin)
make deb             # Alias de package
```

## Licença

Este projeto está licenciado sob a GPL-3.0 — ver [LICENSE](LICENSE) para detalhes.
