#!/usr/bin/env python

#python3 change_delay.py miner 50ms bp 50ms bpminer 50ms

import yaml
import os, sys, getopt

delay={}
cur_path = os.path.dirname(os.path.realpath(__file__))

def load_template():
    templete_yaml_file = os.path.join(cur_path, "conf/gnte.yaml")
    f = open(templete_yaml_file, 'r', encoding='utf-8')
    templete_yaml_raw = f.read()
    templete_yaml = yaml.full_load(templete_yaml_raw)
    return templete_yaml

def change_delay(templete):
    #print(templete_yaml["network"])
    for group in templete["group"]:
        for name in delay:
            if group["name"] == name and delay[name] != "":
                group["delay"] = delay[name]
    for network in templete["network"]:
        for name in delay:
            if network["name"] == name and delay[name] != "":
                network["delay"] = delay[name]
    return templete

def print_yaml(out, case_name):
    out_yaml_file = os.path.join(cur_path, "conf/"+case_name+".yaml")
    f = open(out_yaml_file, 'w', encoding='utf-8')
    yaml.safe_dump(out, f)

def main(argv):
    try:
        opts, _ = getopt.getopt(argv,"h",["name=", "bp=","miner=","client=","bpminer=","bpclient=","minerclient="])
    except getopt.GetoptError:
        print('change_delay.py -bp "xxms" -miner "xxms"...')
        sys.exit(2)
    for opt, arg in opts:
        if opt == '-h':
            print('change_delay.py -bp "xxms" -miner "xxms"...')
            sys.exit()
        elif opt in ("--name"):
            case_name = arg
        elif opt in ("--bp"):
            delay["bp"] = arg
        elif opt in ("--miner"):
            delay["miner "] = arg
        elif opt in ("--client"):
            delay["client"] = arg
        elif opt in ("--bpminer"):
            delay["bpminer "] = arg
        elif opt in ("--bpclient"):
            delay["bpclient "] = arg
        elif opt in ("--minerclient"):
            delay["minerclient"] = arg

    templete = load_template()
    out = change_delay(templete)
    print_yaml(out, case_name)

if __name__ == "__main__":
   main(sys.argv[1:])

