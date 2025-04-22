import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:front_tfg/pantalla_rfb.dart';
import 'package:http/http.dart' as http;

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

    var body = json.encode({'code_qr': widget.qr});
    try {
      var response = await http.post(
        url,
        headers: {'Content-Type': 'application/json'},
        body: body,
      );

      if (response.statusCode == 200) {
        setState(() {
          Navigator.push(
            context,
            MaterialPageRoute(
              builder: (context) => PantallaRfb(),
            ),
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
      appBar: AppBar(title: const Text("Resultado de Análisis")),
      body:
          analizadores.isEmpty
              ? const Center(child: Text("No hay datos"))
              : Column(
                children: [
                  Expanded(
                    child: ListView.builder(
                      itemCount: analizadores.length,
                      itemBuilder: (context, index) {
                        return ListTile(
                          leading: const Icon(Icons.analytics),
                          title: Text(analizadores[index]["analizador"]!),
                          subtitle: Text(analizadores[index]["resultado"]!),
                        );
                      },
                    ),
                  ),
                  Padding(
                    padding: const EdgeInsets.all(16.0),
                    child: Center(
                      child: ElevatedButton(
                        onPressed: () => consultaSegura(),
                        child: const Text('Consulta segura'),
                      ),
                    ),
                  ),
                ],
              ),
    );
  }
}
