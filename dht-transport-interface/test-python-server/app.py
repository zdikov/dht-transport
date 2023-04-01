from flask import Flask, request, Response
import json

app = Flask(__name__)

dht_content = dict()

@app.route('/api/v1/put/', methods=['POST'])
def put_key():
    if request.json['key'] in dht_content.keys():
        return Response("Key already exists", status=403)
    dht_content[request.json['key']] = request.json['value']
    return Response("OK", status=200)

@app.route('/api/v1/getMany/')
def get_keys():
    prefix = request.args.get('prefix')
    result = []
    for (k, v) in dht_content.items():
        if k.startswith(prefix):
            result.append({'key': k, 'value': v})
    return Response(bytes(json.dumps(result), 'utf-8'), status=200)

if __name__ == '__main__':
    app.run()