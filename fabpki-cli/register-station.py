"""
    The BlockMeter Experiment
    ~~~~~~~~~
    This module is necessary to register a meter in the blockchain. It
    receives the meter ID and its respective public key.
    This module must be called before any query against the ledger.
        
    :copyright: © 2020 by Wilson Melo Jr.
"""

import sys
from hfc.fabric import Client as client_fabric
import asyncio

domain = "ptb.de" #you can change for "inmetro.br"
channel_name = "nmi-channel"
cc_name = "fabpki"
cc_version = "1.0"

if __name__ == "__main__":

    #test if the station info was informed as argument
    if len(sys.argv) > 5: # o primeiro  argumentosempre vai ser o chamado do python
        print("Usage:",sys.argv[0], "<station ID> <temperature> <windspeed> <insolation>")
        exit(1)

    #get the station info
    station_id = sys.argv[1]
    temperature = sys.argv[2]
    windspeed = sys.argv[3]
    insolation = sys.argv[4]

    #format the name of the expected public key
    # pub_key_file = meter_id + ".pub"

    #try to retrieve the public key
    '''
    try:
        with open(pub_key_file, 'r') as file:
            pub_key = file.read()
    except:
        print("I could not find a valid public key to the meter",meter_id)
        exit(1)
    '''

    #shows the meter public key
    # print("Continuing with the public key:\n",pub_key)

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
        fcn='registerStation', 
        args=[station_id, temperature, windspeed, insolation], 
        cc_pattern=None))

    #so far, so good
    print("Success on register station and data!")
