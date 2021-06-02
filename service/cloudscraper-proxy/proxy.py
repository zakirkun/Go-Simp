from flask import Flask,request,Response
import cloudscraper

scraper = cloudscraper.create_scraper()
app = Flask(__name__)

@app.route("/")
def hello_world():
    url = request.args.get('URL')
    if url != None:
        result = scraper.get(url)
        return Response(result,content_type="application/json; charset=utf-8")
    return "<p>Hello, World!</p>"