import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:front_tfg/pantalla_rfb.dart';
import 'package:http/http.dart' as http;
import 'package:android_id/android_id.dart';

class AnalisisQr extends StatefulWidget {
  final String analisis;
  final String qr;

  const AnalisisQr({super.key, required this.analisis, required this.qr});

  @override
  _AnalisisQrState createState() => _AnalisisQrState();
}

class _AnalisisQrState extends State<AnalisisQr> {
  List<Map<String, String>> analizadores = [];

  @override
  void initState() {
    super.initState();
    procesarAnalisis();
  }

  void procesarAnalisis() {
    try {
      final List<dynamic> datosJson = jsonDecode(widget.analisis);

      setState(() {
        analizadores =
            datosJson.map<Map<String, String>>((item) {
              return {
                "analizador": item["analizador"].toString(),
                "resultado": item["resultado"].toString(),
              };
            }).toList();
      });
    } catch (e) {
      print("Error al decodificar los datos: $e");
    }
  }

  void consultaSegura() async {
    print("Entrando babyyyyy");
    var url = Uri.parse('http://192.168.1.46:80/consulta_segura');

    final androidIdPlugin = AndroidId();
    final androidId = await androidIdPlugin.getId();

    var body = json.encode({'android_id': androidId, 'qr_code': widget.qr});

    try {
      var response = await http.post(
        url,
        headers: {'Content-Type': 'application/json'},
        body: body,
      );

      if (response.statusCode == 201) {
        int puerto = json.decode(response.body);

        print(puerto);

        setState(() {
          Navigator.push(
            context,
            MaterialPageRoute(builder: (context) => PantallaRfb(port: puerto)),
          );
        });
      } else {
        print('Error wacho: ${response.statusCode}');
      }
    } catch (e) {
      print('Request failed: $e');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFFE3F2FD), // Fondo azul claro
      appBar: AppBar(
        title: const Text(
          "Resultado de Análisis",
          style: TextStyle(color: Colors.white),
        ),
        backgroundColor: const Color(0xFF1976D2), // Azul medio
        iconTheme: const IconThemeData(color: Colors.white),
      ),
      body:
          analizadores.isEmpty
              ? const Center(
                child: Text(
                  "No hay datos",
                  style: TextStyle(fontSize: 18, color: Colors.black54),
                ),
              )
              : Column(
                children: [
                  Card(
                    color: Colors.white,
                    margin: const EdgeInsets.all(16),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(12),
                    ),
                    elevation: 6,
                    child: Padding(
                      padding: const EdgeInsets.all(16.0),
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          const Text(
                            'Contenido del QR:',
                            style: TextStyle(
                              fontSize: 18,
                              fontWeight: FontWeight.bold,
                              color: Colors.black87,
                            ),
                          ),
                          const SizedBox(height: 12),
                          Text(
                            widget.qr,
                            style: const TextStyle(
                              fontSize: 16,
                              color: Colors.black87,
                            ),
                            textAlign: TextAlign.center,
                          ),
                        ],
                      ),
                    ),
                  ),
                  Expanded(
                    child: ListView.builder(
                      itemCount: analizadores.length,
                      itemBuilder: (context, index) {
                        return Card(
                          margin: const EdgeInsets.symmetric(
                            horizontal: 16,
                            vertical: 6,
                          ),
                          shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(10),
                          ),
                          elevation: 2,
                          child: ListTile(
                            leading: Icon(
                              Icons.analytics,
                              color: ((analizadores[index]["resultado"] ?? "") == "No peligroso") ? Colors.green : Colors.red,
                            ),
                            title: Text(
                              analizadores[index]["analizador"] ?? "Desconocido",
                              style: const TextStyle(color: Colors.black87),
                            ),
                            subtitle: Text(
                              analizadores[index]["resultado"] ?? "Sin resultados",
                              style: const TextStyle(color: Colors.black54),
                            ),
                          ),
                        );
                      },
                    ),
                  ),
                  Padding(
                    padding: const EdgeInsets.all(16.0),
                    child: SizedBox(
                      width: double.infinity,
                      child: ElevatedButton(
                        style: ElevatedButton.styleFrom(
                          backgroundColor: const Color(0xFF1976D2),
                          foregroundColor: Colors.white,
                          shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(10),
                          ),
                          padding: const EdgeInsets.symmetric(vertical: 14),
                        ),
                        onPressed: () => consultaSegura(),
                        child: const Text(
                          'Consulta segura',
                          style: TextStyle(fontSize: 16),
                        ),
                      ),
                    ),
                  ),
                ],
              ),
    );
  }
}
