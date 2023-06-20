# Puxar dados de clima do website OpenWeatherMap
# A função recebe como parametro o nome da cidade, e envia ao...
# ...Chaincode o nome da cidade, descrição do clima e temperatura
import requests
import sys
from hfc.fabric import Client as client_fabric
import asyncio

domain = "ptb.de" #you can change for "inmetro.br"
channel_name = "nmi-channel"
cc_name = "fabpki"
cc_version = "1.0"

# Início da API que puxa dados do website OpenWeatherMap
API_KEY = "f0d1f3f95a0cea3c760257aa9379b5d1"

if __name__ == "__main__":

    #test if the city name was informed as argument
    if len(sys.argv) != 2: # o primeiro  argumentosempre vai ser o chamado do python
        print("Usage:",sys.argv[0], "<\"city name\"> ")
        exit(1)

    #get city name from args and send to the API
    cidade = sys.argv[1]
    link = f"https://api.openweathermap.org/data/2.5/weather?q={cidade}&appid={API_KEY}&lang=pt_br&units=metric"

    #request weather info from API
    try:
        requisicao = requests.get(link)
        requisicao.raise_for_status()  # Lança uma exceção caso ocorra um erro HTTP

        requisicao_dic = requisicao.json()

        situacao = requisicao_dic['weather'][0]['description']
        temperatura = requisicao_dic['main']['temp']

        # Converter temperatura de float para string
        temperaturaString = str(temperatura)

        print(f"Descrição do clima em {cidade}")
        print(f"Situação: {situacao}")
        print(f"Temperatura: {temperatura}ºC")

    except requests.exceptions.RequestException as e:
        print("Ocorreu um erro durante a requisição HTTP:", e)
    
    except KeyError as e:
        print("A resposta da API não contém os dados esperados:", e)

    except Exception as e:
        print("Ocorreu um erro:", e)

    #creates a loop object to manage async transactions
    loop = asyncio.get_event_loop()

    #instantiate the hyperledeger fabric client
    c_hlf = client_fabric(net_profile=(domain + ".json"))

    #get access to Fabric as Admin user
    admin = c_hlf.get_user(domain, 'Admin')
    callpeer = "peer0." + domain

    #query peer installed chaincodes, make sure the chaincode is installed
    print("Checking if the chaincode fabpki is properly installed:")
    response = loop.run_until_complete(c_hlf.query_installed_chaincodes(
        requestor=admin,
        peers=[callpeer]
    ))
    print(response)

    #the Fabric Python SDK do not read the channel configuration, we need to add it mannually'''
    c_hlf.new_channel(channel_name)

    #invoke the chaincode to register the meter
    response = loop.run_until_complete(c_hlf.chaincode_invoke(
        requestor=admin, 
        channel_name=channel_name, 
        peers=[callpeer],
        cc_name=cc_name, 
        cc_version=cc_version,
        fcn='registerWeatherFromWeb',
        args=[cidade, situacao, temperaturaString],
        cc_pattern=None))

    #so far, so good
    print("Success on register station and data!")
