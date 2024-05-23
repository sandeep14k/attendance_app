import sys
import json
import face_recognition
from pymongo import MongoClient

def encode_face(image_path):
    # Load the uploaded image file
    
    image = face_recognition.load_image_file(image_path)
    face_encodings = face_recognition.face_encodings(image)

    if not face_encodings:
        return json.dumps({"error": "No face detected"})

    # Assume the first face in the image is the student's face
    student_face_encoding = face_encodings[0].tolist()
    return json.dumps({"face_encoding": student_face_encoding})

if __name__ == "__main__":
    image_path = sys.argv[1]
    result = encode_face(image_path)
    print(result)
