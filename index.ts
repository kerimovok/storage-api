import multer from "multer";
import express from "express";
import path from "path";
import bodyParser from "body-parser";
import cors from "cors";
import fs from "fs";

const app = express();
const port = 3002;
app.use(bodyParser.json());
app.use(cors());

const storage = multer.diskStorage({
  destination: (req, file, cb) => {
    const date = new Date();
    const year = date.getFullYear().toString();
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");
    const extension = path.extname(file.originalname).slice(1);
    const uploadPath = path.join("uploads", year, month, day, extension);

    fs.mkdirSync(uploadPath, { recursive: true });
    cb(null, uploadPath);
  },
  filename: (req, file, cb) => {
    cb(null, Date.now() + path.extname(file.originalname));
  },
});

const upload = multer({ storage: storage });

app.post("/upload", upload.single("image"), (req, res) => {
  if (!req.file) {
    return res.status(400).send("No file uploaded.");
  }
  const imageUrl = `/${req.file.path}`;
  res.send({ imageUrl });
});

app.use("/uploads", express.static(path.join(__dirname, "uploads")));

app.listen(port, () => {
  console.log(`Server is running on http://0.0.0.0:${port}`);
});
