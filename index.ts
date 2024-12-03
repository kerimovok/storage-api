import multer from 'multer'
import express from 'express'
import path from 'path'
import bodyParser from 'body-parser'
import cors from 'cors'
import fs from 'fs'

const app = express()
const port = process.env.PORT || 3003
app.use(bodyParser.json())
app.use(cors())

const storage = multer.diskStorage({
	destination: (req, file, cb) => {
		const date = new Date()
		const year = date.getFullYear().toString()
		const month = String(date.getMonth() + 1).padStart(2, '0')
		const day = String(date.getDate()).padStart(2, '0')
		const extension = path.extname(file.originalname).slice(1)
		const uploadPath = path.join('uploads', year, month, day, extension)

		fs.mkdirSync(uploadPath, { recursive: true })
		cb(null, uploadPath)
	},
	filename: (req, file, cb) => {
		cb(null, Date.now() + path.extname(file.originalname))
	},
})

const upload = multer({ storage: storage })

const v1Router = express.Router()

v1Router.post('/upload.single', upload.single('image'), (req, res) => {
	if (!req.file) {
		return res.status(400).send('No file uploaded.')
	}
	const imageUrl = `/${req.file.path}`
	res.send({ imageUrl })
})

v1Router.post('/upload.multiple', upload.array('images'), (req, res) => {
	if (!req.files) {
		return res.status(400).send('No files uploaded.')
	}
	const imageUrls = (req.files as Express.Multer.File[]).map(
		(file) => `/${file.path}`
	)
	res.send({ imageUrls })
})

// Static file serving
v1Router.use('/uploads', express.static(path.join(__dirname, 'uploads')))

app.use('/api/v1', v1Router)

app.listen(port, () => {
	console.log(`Server is running on http://0.0.0.0:${port}`)
})
