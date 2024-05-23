import sys
import json
import face_recognition
import numpy as np
from pymongo import MongoClient

def recognize_face(image_path):
    client = MongoClient('mongodb://localhost:27017')
    db = client['Attendance_App']
    students_collection = db['student_login_detail']

    # Load the uploaded image file
    image = face_recognition.load_image_file(image_path)

    # Get the face encodings for the uploaded image
    face_encodings = face_recognition.face_encodings(image)

    if not face_encodings:
        return json.dumps({"error": "No face detected"})

    # Assume the first face is the student to be recognized
    uploaded_face_encoding = face_encodings[0]

    # Fetch all students and their face encodings from the database
    students = students_collection.find()
    for student in students:
        known_face_encoding = np.array(student['face_encoding'])
        # Compare the uploaded face with the known face
        matches = face_recognition.compare_faces([known_face_encoding], uploaded_face_encoding)
        if True in matches:
            return json.dumps({"student_id": student['user_id']})

    return json.dumps({"error": "No matching student found"})

if __name__ == "__main__":
    image_path = sys.argv[1]
    result = recognize_face(image_path)
    print(result)
