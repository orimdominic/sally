import http from "node:http";
import { pipeline } from "@huggingface/transformers";

async function getRequestBody(req) {
  return new Promise((resolve, reject) => {
    let data = "";
    req.on("data", (chunk) => (data += chunk));
    req.on("end", () => resolve(JSON.parse(data)));
    req.on("error", reject);
  });
}

const extractor = await pipeline(
  "feature-extraction",
  "Xenova/all-MiniLM-L6-v2",
);

const server = http.createServer(async function (req, res) {
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.setHeader("Access-Control-Allow-Methods", "POST, OPTIONS");
  res.setHeader("Access-Control-Allow-Headers", "Content-Type");

  switch (req.method) {
    case "POST": {
      const body = await getRequestBody(req);
      const output = await extractor(body.content, {
        pooling: "mean",
        normalize: true,
      });

      res.end(JSON.stringify({ embeddings: Array.from(output.data) }));
      return;
    }

    default: {
      return res.end("non-post request received");
    }
  }
});

const port = Number(process.env.PORT) || 3333;
server.listen(port, function () {
  console.log(`Embedding server running on :${port}`);
});
