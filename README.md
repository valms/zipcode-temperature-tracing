# Rastreamento de Temperatura por CEP

Este projeto faz parte dos Laboratórios de Pós-Graduação em Go da FullCycle. Consiste em microsserviços para obtenção de
informações meteorológicas baseadas em CEP, utilizando OpenTelemetry para rastreamento.

## Desafio

### Objetivo

Desenvolver um sistema em Go que receba um CEP, identifica a cidade e retorna o clima atual (temperatura em graus
celsius, fahrenheit e kelvin) juntamente com a cidade. Esse sistema deverá implementar OTEL(Open Telemetry) e Zipkin.

### Requisitos - Serviço A (responsável pelo input):

- Receber um input de 8 dígitos via POST, através do schema: `{ "cep": "29902555" }`
- Validar se o input é válido (contém 8 dígitos) e é uma STRING
- Se válido, encaminhar para o Serviço B via HTTP
- Se inválido, retornar:
	- Código HTTP: 422
	- Mensagem: "invalid zipcode"

### Requisitos - Serviço B (responsável pela orquestração):

- Receber um CEP válido de 8 dígitos
- Realizar a pesquisa do CEP e encontrar o nome da localização
- Retornar as temperaturas formatadas em: Celsius, Fahrenheit, Kelvin juntamente com o nome da localização
- Cenários de resposta:
	- Sucesso:
		- Código HTTP: 200
		- Response Body:
	  ``` json
	  {"city: "São Paulo", "temp_C": 28.5, "temp_F": 28.5, "temp_K": 28.5 }
	  ```
	- CEP inválido (com formato correto):
		- Código HTTP: 422
		- Mensagem: "invalid zipcode"
	- CEP não encontrado:
		- Código HTTP: 404
		- Mensagem: "can not find zipcode"

### Implementação OTEL + Zipkin:

- Implementar tracing distribuído entre Serviço A - Serviço B
- Utilizar span para medir o tempo de resposta do serviço de busca de CEP e busca de temperatura

## Estrutura do Projeto

```
.
├── config
│   └── otel
├── serviceA
├── serviceB
├── .editorconfig
├── .gitignore
├── README.md
└── docker-compose.yml
```

## Pré-requisitos

- Docker
- Docker Compose

## Como Executar o Projeto em Ambiente de Desenvolvimento

1. Clone o repositório:
   ```
   git clone https://github.com/valms/zipcode-temperature-tracing.git
   cd zipcode-temperature-tracing
   ```

2. Configure as variáveis de ambiente:
   Crie um arquivo `.env` no diretório raiz e adicione as seguintes variáveis:
   ```
   API_KEY=sua_weather_api_key
   ```
   Substitua `sua_weather_api_key` por uma chave API válida do API WeatherAPI.

3. Inicie os serviços usando Docker Compose:
   ```
   docker-compose up --build
   ```

4. Os serviços estarão disponíveis nos seguintes endereços:
	- ServiceA: http://localhost:8080
	- ServiceB: http://localhost:8081
	- Coletor OpenTelemetry: http://localhost:4318 (Exportador OTLP HTTP)

## Uso

Para utilizar o serviço, envie uma requisição POST para o ServiceA com um CEP no corpo da requisição:

```
curl -X POST http://localhost:8080 -H "Content-Type: application/json" -d '{"cep": "01001000"}'
```

O serviço retornará as informações de temperatura para o CEP fornecido.

## Rastreamento

Este projeto utiliza OpenTelemetry para rastreamento distribuído. Você pode visualizar os rastros usando Zipkin.

## Serviços

### ServiceA

Ponto de entrada da aplicação. Recebe requisições de CEP, valida-as e as encaminha para o ServiceB.

### ServiceB

Recebe requisições de CEP do ServiceA, busca as informações da cidade e obtém as informações de temperatura.

## Configuração

O diretório `config/otel` contém arquivos de configuração para o Coletor OpenTelemetry.

# Fontes

1. https://opentelemetry.io/docs/languages/go/instrumentation/
2. https://github.com/open-telemetry/opentelemetry-go/issues/23
3. https://opentelemetry.io/docs/languages/go/getting-started/
4. https://stackoverflow.com/questions/72867313/opentelemetry-open-span-in-one-process-and-close-it-in-another
5. https://opentelemetry.io/docs/concepts/signals/traces/
