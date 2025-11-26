# Guia do Proxima CLI

Proxima é uma ferramenta CLI moderna para gerenciar VMs Proxmox com estrutura de comandos simplificada e capacidades SSH. Este guia cobre instalação, configuração e uso.

## Sumário

- [Instalação](#instalação)
- [Configuração](#configuração)
- [Estrutura de Comandos](#estrutura-de-comandos)
- [Modos de Operação](#modos-de-operação)
- [Comandos Disponíveis](#comandos-disponíveis)
- [Configuração SSH](#configuração-ssh)
- [Configuração de Rede](#configuração-de-rede)
- [Shutdown Graceful vs Imediato](#shutdown-graceful-vs-imediato)
- [Exemplos](#exemplos)
- [Testes](#testes)
- [Solução de Problemas](#solução-de-problemas)

## Instalação

1. Clone o repositório:
```bash
git clone <url-do-repositório>
cd proxima
```

2. Compile o binário:
```bash
cd src
go build -o ../proxima ./cmd/main.go
```

3. Torne-o executável:
```bash
chmod +x proxima
```

## Configuração

Proxima usa um arquivo de configuração YAML (`config.yaml`) para definir configurações de conexão Proxmox e configurações de VM.

### Estrutura Básica de Configuração

```yaml
proxmox:
  host: "192.168.1.100"
  port: 8006
  user: "root@pam"
  password: "sua_senha"
  node: "pve"

ssh:
  user: "root"
  password: "ssh_password"
  key_path: "/caminho/para/chave/privada"
  port: 22
  copy_local_key: false

vms:
  - name: "web-server-01"
    vmid: 100
    cores: 2
    memory: 2048
    disk_size: "20G"
    network:
      bridge: "vmbr0"
      vlan: 100
      model: "virtio"
    os_template: "local:vztmpl/ubuntu-22.04-standard_22.04-1_amd64.tar.zst"
    tags: ["web", "production"]
    auto_start: true
    ssh:
      user: "ubuntu"
      password: "ubuntu_password"
      key_path: "/caminho/para/chave/vm"
      copy_local_key: true
      authorized_keys:
        - "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC..."
    scripts:
      - name: "install-nginx"
        path: "/scripts/install-nginx.sh"
        args: []
        timeout: 300
```

### Campos de Configuração

#### Seção Proxmox
- `host`: IP/hostname do servidor Proxmox
- `port`: Porta da API Proxmox (padrão: 8006)
- `user`: Usuário Proxmox (ex: "root@pam")
- `password`: Senha Proxmox
- `node`: Nome do nó Proxmox

#### Seção SSH
- `user`: Usuário SSH padrão
- `password`: Senha SSH padrão
- `key_path`: Caminho para chave SSH privada (opcional)
- `port`: Porta SSH (padrão: 22)
- `copy_local_key`: Se deve copiar `~/.ssh/id_rsa.pub` local para as VMs

#### Configuração da VM
- `name`: Nome da VM
- `vmid`: ID único da VM
- `cores`: Número de núcleos de CPU
- `memory`: RAM em MB
- `disk_size`: Tamanho do disco (ex: "20G")
- `network`: Configuração de rede
  - `bridge`: Bridge de rede
  - `vlan`: ID da VLAN
  - `model`: Modelo de rede (ex: "virtio")
- `os_template`: Caminho do template de SO
- `tags`: Lista de tags para organização
- `auto_start`: Se deve iniciar VM após criação
- `ssh`: Configurações SSH específicas da VM
- `scripts`: Scripts para executar após criação da VM

## Estrutura de Comandos

Proxima usa uma estrutura de comandos simplificada:

```bash
proxima <host> <comando> [argumentos]
proxima <yaml> <comando> [argumentos]
proxima help <comando>
```

### Mudanças Principais
- **Argumentos Posicionais**: VMID agora é posicional, não flag
- **Sem Flags de Help**: Sem flags `-h`/`--help`, use `proxima help <comando>`
- **Sintaxe Simplificada**: `proxima <host> start 100` em vez de `proxima <host> start --vmid 100`

## Modos de Operação

### 1. Modo Host (SSH Direto)
```bash
proxima 10.13.250.11 list
proxima 10.13.250.11 start 100
proxima 10.13.250.11 shutdown 100
proxima 10.13.250.11 stop 100
proxima 10.13.250.11 delete 100
```
- Conexão SSH direta usando `~/.ssh/id_rsa`
- Sem prompts de senha
- Método mais rápido para operações do dia a dia
- Usa comandos `qm` via SSH

### 2. Modo Arquivo de Configuração
```bash
proxima config.yaml create --name web-server
proxima config.yaml
```
- Usa arquivo de configuração YAML
- Criação idempotente de VM (só cria se não existe)
- Abordagem de Infrastructure as Code

### 3. Modo Login Interativo
```bash
proxima --login 10.13.250.11 list
```
- Solicita nome de usuário e senha SSH
- Útil quando chaves SSH não estão configuradas
- Método de autenticação fallback

## Comandos Disponíveis

### Comandos Básicos

#### Listar VMs
```bash
# Modo host
proxima 10.13.250.11 list

# Modo arquivo config
proxima config.yaml

# Modo interativo
proxima --login 10.13.250.11 list
```

#### Ciclo de Vida da VM
```bash
# Iniciar VM (VMID posicional)
proxima 10.13.250.11 start 100

# Shutdown graceful VM (VMID posicional)
proxima 10.13.250.11 shutdown 100

# Parada imediata VM (VMID posicional)
proxima 10.13.250.11 stop 100

# Deletar VM (VMID posicional)
proxima 10.13.250.11 delete 100

# Criar VM do config
proxima config.yaml create --name web-server
```

### Sistema de Help
```bash
# Mostrar visão geral
proxima

# Mostrar help específico do comando
proxima help list
proxima help start
proxima help shutdown
proxima help stop
proxima help delete
proxima help create
```

### Opções de Comando

- `--login`: Usar login interativo em vez de chaves SSH
- `--name <name>`: Nome da VM para criação (flag)
- VMID agora é posicional, não flag

## Configuração SSH

### Gerenciamento de Chaves SSH

Proxima suporta múltiplos métodos de autenticação SSH:

1. **Caminho de Chave Explícito**: Especifique `key_path` na configuração
2. **Fallback de Chave Padrão**: Se `key_path` não for especificado, Proxima busca por:
   - `~/.ssh/id_ed25519`
   - `~/.ssh/id_rsa`
   - `~/.ssh/id_ecdsa`
   - `~/.ssh/id_dsa`
3. **Autenticação por Senha**: Use campo `password`
4. **Cópia de Chave**: Defina `copy_local_key: true` para copiar chave pública local

### Recurso de Cópia de Chave SSH

Quando `copy_local_key: true` está habilitado:
- Proxima copia `~/.ssh/id_rsa.pub` da máquina local
- Adiciona ao `~/.ssh/authorized_keys` na VM
- Cria diretório `.ssh` se não existir
- Define permissões adequadas (600 para authorized_keys)

### Configurações SSH Específicas por VM

Cada VM pode ter sua própria configuração SSH que sobrepõe as configurações globais:

```yaml
ssh:
  user: "ubuntu"           # Usuário específico da VM
  password: "vm_password"  # Senha específica da VM
  key_path: "/caminho/para/chave/vm"  # Chave específica da VM
  copy_local_key: true     # Copiar chave local para esta VM
  authorized_keys:         # Chaves pré-autorizadas
    - "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC..."
```

## Exemplos

### Exemplo 1: Servidor Web Básico

```yaml
vms:
  - name: "web-server-01"
    vmid: 100
    cores: 2
    memory: 2048
    disk_size: "20G"
    network:
      bridge: "vmbr0"
      vlan: 100
      model: "virtio"
    os_template: "local:vztmpl/ubuntu-22.04-standard_22.04-1_amd64.tar.zst"
    tags: ["web", "production"]
    auto_start: true
    ssh:
      user: "ubuntu"
      copy_local_key: true
    scripts:
      - name: "install-nginx"
        path: "/scripts/install-nginx.sh"
        timeout: 300
```

### Exemplo 2: Servidor de Banco de Dados com SSH Personalizado

```yaml
vms:
  - name: "database-01"
    vmid: 101
    cores: 4
    memory: 4096
    disk_size: "50G"
    network:
      bridge: "vmbr0"
      vlan: 200
      model: "virtio"
    os_template: "local:vztmpl/ubuntu-22.04-standard_22.04-1_amd64.tar.zst"
    tags: ["database", "production"]
    ssh:
      user: "ubuntu"
      password: "senha_segura_bd"
      key_path: "/keys/chave_database"
      copy_local_key: false
    scripts:
      - name: "install-postgresql"
        path: "/scripts/install-postgresql.sh"
        args: ["--version=14"]
        timeout: 600
```

### Exemplo 3: Ambiente de Desenvolvimento

```yaml
ssh:
  user: "developer"
  copy_local_key: true  # Copiar chave de dev para todas as VMs

vms:
  - name: "dev-env-01"
    vmid: 200
    cores: 2
    memory: 4096
    disk_size: "30G"
    network:
      bridge: "vmbr0"
      vlan: 300
      model: "virtio"
    os_template: "local:vztmpl/ubuntu-22.04-standard_22.04-1_amd64.tar.zst"
    tags: ["development"]
    auto_start: false
    ssh:
      user: "developer"
      copy_local_key: true
    scripts:
      - name: "setup-dev-tools"
        path: "/scripts/setup-dev-tools.sh"
        timeout: 600
```

## Dicas e Melhores Práticas

1. **Segurança**: Use chaves SSH em vez de senhas quando possível
2. **Organização**: Use tags para categorizar VMs (ex: "web", "database", "development")
3. **Backup**: Mantenha seu arquivo de configuração em controle de versão
4. **Testes**: Teste configurações de VM em ambiente de desenvolvimento primeiro
5. **Idempotência**: Use modo arquivo config para infraestrutura reproduzível
6. **Estratégia de Shutdown**: Use `shutdown` para VMs de produção, `stop` para desenvolvimento
7. **Estrutura de Comandos**: Lembre-se da nova sintaxe: `proxima <host> <comando> <vmid>`

## Configuração de Rede

### Suporte a VLAN

Proxima agora suporta tagging de VLAN para interfaces de rede de VM:

```yaml
vms:
  - name: "web-server-01"
    vmid: 100
    network:
      bridge: "vmbr0"
      vlan: 100        # ID da VLAN
      model: "virtio"  # Modelo de rede
```

### Resolução Dinâmica de IP

Proxima resolve automaticamente endereços IP de VM usando:

1. **QEMU Agent**: Método principal para descoberta de IP real
2. **Padrão Fallback**: `192.168.1.{vmid+100}` se QEMU Agent indisponível

## Shutdown Graceful vs Imediato

Proxima fornece dois métodos diferentes de shutdown:

### Shutdown Graceful
```bash
proxima 10.13.250.11 shutdown 100
```

**Recursos:**
- Envia sinal de shutdown ACPI para VM
- Espera VM desligar gracefulmente
- Lida com estado de startup da VM (espera se VM está starting)
- Timeout: 60 segundos para conclusão do shutdown
- Timeout: 120 segundos para conclusão do startup da VM
- Feedback em tempo real

**Processo:**
1. Verifica status da VM
2. Se VM está starting, espera até 2 minutos para fully start
3. Envia comando graceful shutdown
4. Monitora status por até 60 segundos
5. Reporta sucesso ou timeout

**Casos de Uso:**
- Ambientes de produção onde integridade de dados é crítica
- VMs rodando databases ou aplicações que precisam de shutdown limpo
- Janelas de manutenção programadas

### Parada Imediata
```bash
proxima 10.13.250.11 stop 100
```

**Recursos:**
- Para VM imediatamente (equivalente a power off)
- Sem período de espera
- Terminação force
- Execução rápida

**Casos de Uso:**
- Ambientes de desenvolvimento
- VMs não responsivas
- Situações de emergência
- Ciclos de restart rápidos

### Tratamento de Erros

**Erros de Comando Shutdown:**
- `VM quit/powerdown failed - got timeout`: VM não respondeu ao sinal de shutdown
- `timeout waiting for VM to fully start`: VM demorou demais para iniciar antes do shutdown
- `timeout waiting for VM to shutdown`: VM não desligou dentro de 60 segundos

**Soluções:**
- Use comando `stop` para terminação imediata
- Verifique logs da VM para problemas de shutdown
- Garanta que QEMU Agent está instalado e rodando
- Verifique se o SO da VM está respondendo a sinais ACPI

## Testes

### Executando Testes

```bash
cd src
go test ./... -v
```

### Cobertura de Testes

O projeto inclui testes unitários abrangentes para:
- Validação de configuração
- Entidades de domínio (VM, Command)
- Funcionalidade do adaptador SSH
- Cenários de tratamento de erros

### Exemplos de Testes

```bash
# Executar pacote de teste específico
go test ./internal/adapters/config -v

# Executar com cobertura
go test ./... -cover

# Executar teste específico
go test ./internal/core/domain -run TestNewVM
```

## Solução de Problemas

### Problemas Comuns

1. **Conexão SSH Falhou**:
   - Verifique credenciais SSH e caminhos de chaves
   - Verifique conectividade de rede
   - Certifique-se de que o serviço SSH está rodando na VM
   - Verifique se QEMU Agent está instalado e rodando

2. **Criação de VM Falhou**:
   - Verifique credenciais Proxmox e configurações TLS
   - Verifique se ID da VM já está em uso
   - Certifique-se de que o template de SO existe
   - Valide configuração de VLAN

3. **Problemas de Resolução de IP**:
   - Certifique-se de que QEMU Agent está instalado na VM
   - Verifique se a rede da VM está configurada corretamente
   - Verifique configurações de bridge e VLAN

4. **Execução de Script Falhou**:
   - Verifique permissões e caminhos do arquivo de script
   - Verifique sintaxe do script
   - Certifique-se de que a VM está acessível via SSH
   - Verifique configurações de timeout do script

### Mensagens de Erro Aprimoradas

Proxima agora fornece mensagens de erro detalhadas com:
- Contexto específico de falha
- Sugestões de solução
- Informações de debug
- Stack traces quando disponível

### Logging

Proxima inclui logging estruturado para:
- Operações da API Proxmox
- Tentativas de conexão SSH
- Validação de configuração
- Eventos do ciclo de vida da VM

## Suporte

Para problemas e solicitações de funcionalidades, por favor consulte o repositório do projeto.