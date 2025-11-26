# Proxima

Ferramenta CLI moderna para gerenciamento de VMs Proxmox com arquitetura limpa e estrutura de comandos simplificada.

## Funcionalidades

- **Estrutura de Comandos Simplificada**: `proxima <host> <comando>` e `proxima <yaml> <comando>`
- **Argumentos Posicionais**: Sem mais flags complexas para VMID - use `proxima <host> start 100`
- **Shutdown Graceful**: Comandos `stop` (imediato) e `shutdown` (graceful)
- **Múltiplos Modos de Operação**: Arquivo config, SSH direto, e login interativo
- **Infraestrutura Idempotente**: Criação de VM com verificação de existência
- **Arquitetura Limpa**: Arquitetura hexagonal com separação adequada de responsabilidades
- **Instalação Silenciosa**: Instalação sem prompts
- **Sem Flags de Help**: Interface limpa sem flags `-h`/`--help`

## Estrutura do Projeto

```
src/
├── cmd/                    # CLI principal
├── internal/
│   ├── core/
│   │   ├── domain/        # Entidades principais
│   │   ├── ports/         # Definições de interface
│   │   └── service/       # Lógica de negócio
│   └── adapters/
│       ├── config/        # Configuração YAML
│       ├── proxmox/       # Integração Proxmox API
│       └── ssh/           # Cliente SSH
└── config.yaml           # Arquivo de configuração
```

## Configuração

Edite o arquivo `config.yaml` com suas configurações do Proxmox e VMs:

```yaml
proxmox:
  host: "192.168.1.100"
  port: 8006
  user: "root@pam"
  password: "your_password"
  node: "pve"

ssh:
  user: "root"
  password: "ssh_password"
  key_path: "/path/to/private/key"
  port: 22

vms:
  - name: "web-server-01"
    vmid: 100
    cores: 2
    memory: 2048
    disk_size: "20G"
    # ... mais configurações
```

## Estrutura de Comandos

### Uso Básico
```bash
proxima <host> <comando> [argumentos]
proxima <yaml> <comando> [argumentos]
proxima help <comando>
```

### Modos de Operação

#### 1. Modo Host (SSH Direto)
```bash
proxima 10.13.250.11 list
proxima 10.13.250.11 start 100
proxima 10.13.250.11 shutdown 100
proxima 10.13.250.11 stop 100
proxima 10.13.250.11 delete 100
```

#### 2. Modo Arquivo de Configuração
```bash
proxima config.yaml create --name web-server
proxima config.yaml
```

#### 3. Modo Login Interativo
```bash
proxima --login 10.13.250.11 list
```

### Comandos Disponíveis

| Comando | Descrição | Exemplo | Argumentos |
|---------|-----------|---------|-----------|
| `list` | Listar todas as VMs | `proxima 10.13.250.11 list` | Nenhum |
| `start <vmid>` | Iniciar VM | `proxima 10.13.250.11 start 100` | ID da VM (posicional) |
| `stop <vmid>` | Parar VM imediatamente | `proxima 10.13.250.11 stop 100` | ID da VM (posicional) |
| `shutdown <vmid>` | Desligar VM gracefulmente | `proxima 10.13.250.11 shutdown 100` | ID da VM (posicional) |
| `delete <vmid>` | Deletar VM | `proxima 10.13.250.11 delete 100` | ID da VM (posicional) |
| `create --name <vm_name>` | Criar VM do config | `proxima config.yaml create --name web` | Nome da VM (flag) |

### Sistema de Help
```bash
proxima                    # Mostrar visão geral
proxima help <comando>     # Mostrar help específico
```

### Compilar
```bash
# Usando Makefile (recomendado)
make build

# Ou manualmente
cd src
go build -o proxima ./cmd
```

### Instalação

#### Instalar no sistema
```bash
make install
```

#### Desinstalar
```bash
make uninstall
```

#### Limpar build
```bash
make clean
```

#### Executar testes
```bash
cd src
go test ./...
```

## Estrutura do Projeto

- **Domain**: Contém entidades principais e lógica de negócio
- **Adapters**: Gerenciam integrações externas (Proxmox API, SSH, Config)
- **Service**: Orquestra operações entre componentes

## Dependências

- `cobra`: CLI framework
- `yaml`: Parser YAML
- `crypto/ssh`: Cliente SSH
- `crypto/tls`: Configuração TLS para HTTPS

## Novas Funcionalidades

### Shutdown Graceful vs Imediato
```bash
# Shutdown graceful (espera VM desligar properly)
proxima 10.13.250.11 shutdown 100

# Parada imediata (force stop)
proxima 10.13.250.11 stop 100
```

O comando `shutdown`:
- Espera VM estar fully running se estiver starting
- Envia sinal de shutdown graceful
- Espera até 60 segundos para VM parar
- Feedback claro durante o processo

### Estrutura de Comandos Simplificada
- Argumentos posicionais para VMID
- Sem flags `-h`/`--help`
- Sintaxe limpa: `proxima <host> <comando> <vmid>`

### Resolução Dinâmica de IP
- Descoberta automática de IP via QEMU Agent
- Fallback para padrões de IP configuráveis
- Sem mais endereços IP hardcoded

### Segurança Aprimorada
- Suporte TLS/SSL para certificados autoassinados
- Múltiplos métodos de autenticação SSH
- Cópia automática de chaves SSH para VMs

### Melhorias de Rede
- Suporte completo a VLAN na configuração de VM
- Configuração flexível de interface de rede
- Personalização de bridge e modelo

### Qualidade e Confiabilidade
- Validação de configuração com mensagens de erro detalhadas
- Cobertura abrangente de testes unitários
- Mensagens de erro aprimoradas com contexto
- Logging estruturado para debugging