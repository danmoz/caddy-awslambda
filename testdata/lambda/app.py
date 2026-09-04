from fastapi import FastAPI, Request
from mangum import Mangum

app = FastAPI()


@app.get("/health")
async def health():
    return {"ok": True}


@app.api_route("/echo", methods=["GET", "POST"])
async def echo(request: Request):
    return {
        "method": request.method,
        "query": dict(request.query_params),
        "headers": {
            "authorization": request.headers.get("authorization"),
            "x-test": request.headers.get("x-test"),
        },
        "body": (await request.body()).decode("utf-8"),
    }


handler = Mangum(app)
