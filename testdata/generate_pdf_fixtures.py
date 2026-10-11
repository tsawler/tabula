"""Regenerate checked-in fixtures: pip install reportlab pypdf cryptography Pillow.
Run from any directory. No Python dependencies are required by go test.
Encryption API: https://pypdf.readthedocs.io/en/6.7.1/user/encryption-decryption.html
"""
from pathlib import Path
from io import BytesIO
from PIL import Image
from reportlab.pdfgen import canvas
from reportlab.lib.utils import ImageReader
from pypdf import PdfReader, PdfWriter

root = Path(__file__).resolve().parent.parent
image = ImageReader(Image.new("RGB", (8, 8), (30, 80, 140)))


def make_pdf(path, placements):
    c = canvas.Canvas(str(path), pagesize=(200, 300), invariant=1)
    for rect in placements:
        if rect is not None:
            c.drawImage(image, *rect)
        else:
            c.drawString(20, 150, "Encrypted Hello World")
        c.showPage()
    c.save()


make_pdf(root / "testdata/illustrations.pdf", [(10, 15, 180, 270)] * 3)
make_pdf(root / "testdata/full_page_images.pdf", [(0, 0, 200, 300)] * 2)
make_pdf(root / "testdata/discrete_figures.pdf", [(20, 30, 40, 60)])
make_pdf(root / "testdata/text_only.pdf", [None])
encrypted = root / "reader/testdata/encrypted"
encrypted.mkdir(parents=True, exist_ok=True)
make_pdf(encrypted / "plain.pdf", [None])
for name, algorithm in [("rc4_40", "RC4-40"), ("rc4_128", "RC4-128"),
                        ("aesv2", "AES-128"), ("aesv3", "AES-256")]:
    writer = PdfWriter(clone_from=PdfReader(encrypted / "plain.pdf"))
    writer.encrypt("", owner_password="", algorithm=algorithm)
    writer.write(encrypted / (name + ".pdf"))
