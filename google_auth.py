import json
import urllib.request
import urllib.parse
from http.server import BaseHTTPRequestHandler, HTTPServer
import threading

CLIENT_ID = "YOUR_CLIENT_ID.apps.googleusercontent.com"
CLIENT_SECRET = "YOUR_CLIENT_SECRET"
REDIRECT_URI = "http://localhost:8080/"
SCOPE = "https://mail.google.com/ https://www.googleapis.com/auth/calendar"

AUTH_URL = f"https://accounts.google.com/o/oauth2/v2/auth?client_id={CLIENT_ID}&redirect_uri={urllib.parse.quote(REDIRECT_URI)}&response_type=code&scope={urllib.parse.quote(SCOPE)}&access_type=offline&prompt=consent"

auth_code = None

class OAuthCallbackHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        global auth_code
        self.send_response(200)
        self.send_header("Content-type", "text/html")
        self.end_headers()
        
        query = urllib.parse.urlparse(self.path).query
        params = urllib.parse.parse_qs(query)
        
        if "code" in params:
            auth_code = params["code"][0]
            self.wfile.write(b"<html><body><h1>Exito!</h1><p>Autorizacion completada. Ya puedes cerrar esta ventana y volver a la terminal.</p></body></html>")
        else:
            self.wfile.write(b"<html><body><h1>Error</h1><p>No se encontro el codigo de autorizacion.</p></body></html>")
            
        # Stop the server in a separate thread so we can send the response first
        threading.Thread(target=self.server.shutdown).start()

    def log_message(self, format, *args):
        pass # Suppress HTTP logs

print("\n" + "="*80)
print("Paso 1: Haz clic (o copia y pega) en la siguiente URL en tu navegador:")
print("="*80)
print(AUTH_URL)
print("="*80)
print("\nEsperando a que des permisos en el navegador...")

server = HTTPServer(("localhost", 8080), OAuthCallbackHandler)
server.serve_forever()

if auth_code:
    print("\nCodigo recibido! Obteniendo Refresh Token...")
    
    token_url = "https://oauth2.googleapis.com/token"
    data = urllib.parse.urlencode({
        "code": auth_code,
        "client_id": CLIENT_ID,
        "client_secret": CLIENT_SECRET,
        "redirect_uri": REDIRECT_URI,
        "grant_type": "authorization_code"
    }).encode("utf-8")
    
    req = urllib.request.Request(token_url, data=data)
    try:
        with urllib.request.urlopen(req) as response:
            result = json.loads(response.read().decode("utf-8"))
            refresh_token = result.get("refresh_token")
            
            if refresh_token:
                print("\n" + "="*80)
                print("¡REFRESH TOKEN OBTENIDO CON EXITO!")
                print(refresh_token)
                print("="*80)
                
                # Update config.json
                with open("config.json", "r") as f:
                    config = json.load(f)
                config["gmail_refresh_token"] = refresh_token
                with open("config.json", "w") as f:
                    json.dump(config, f, indent=4)
                    
                print("\nEl Refresh Token se ha guardado automaticamente en config.json.")
            else:
                print("Error: Google no devolvió un refresh_token. Asegúrate de que estás dando permisos por primera vez (o borra los permisos previos en tu cuenta de Google).")
    except Exception as e:
        print(f"Error al conectar con Google: {e}")
else:
    print("No se recibió ningún código.")
