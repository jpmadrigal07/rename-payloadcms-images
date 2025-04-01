# Rename PayloadCMS Images

This Go program renames image files stored in an AWS S3-compatible service and updates the file names in a MongoDB collection of PayloadCMS. It primarily ensures that the file names follow a consistent dash-separated format instead of containing spaces, underscores, or multiple dashes.

## Features
- Connects to a MongoDB database to retrieve image metadata.
- Renames image files in AWS S3-compatible storage.
- Updates the filenames in the MongoDB collection.
- Supports renaming a single specified media file or multiple files with problematic filenames.

## Setup and Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/rename-payloadcms-images.git
   cd rename-payloadcms-images
   ```
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Create a `.env` file with the required environment variables:
   ```
   MONGO_DB_URI=mongodb://localhost:27017
   MONGO_DB_NAME=yourdbname
   MONGO_DB_COLLECTION=mediacollection
   AWS_BUCKET=yourbucketname
   AWS_REGION=us-east-1
   AWS_ENDPOINT=http://localhost:4566
   AWS_ACCESS_KEY_ID=youraccesskey
   AWS_SECRET_ACCESS_KEY=yoursecretkey
   MEDIA_ID=optionalmediaid
   ```
4. Running the Program
   ```
   go run main.go
   ```

## How It Works
1. Loads environment variables from a `.env` file.
2. Connects to the specified MongoDB instance.
3. Establishes a session with AWS S3-compatible storage.
4. Retrieves media records that either contain problematic characters in the filename or match a specified media ID.
5. Renames the files by replacing spaces, underscores, and multiple dashes with a single dash.
6. Ensures unique filenames by appending a number if a file with the same name already exists.
7. Copies the file to the new key in the storage bucket and deletes the old file.
8. Updates the filename in the MongoDB database.

## Environment Variables
The following environment variables are required:
- `MONGO_DB_URI` - URI for connecting to MongoDB.
- `MONGO_DB_NAME` - Database name.
- `MONGO_DB_COLLECTION` - Collection name for media files.
- `AWS_BUCKET` - Name of the S3 bucket.
- `AWS_REGION` - AWS region.
- `AWS_ENDPOINT` - Custom endpoint for S3-compatible service.
- `AWS_ACCESS_KEY_ID` - AWS access key.
- `AWS_SECRET_ACCESS_KEY` - AWS secret key.
- `MEDIA_ID` - (Optional) Specific media ID to rename. If you want to test it with single media first, just get the `_id` and add it here, otherwise, leave it blank
