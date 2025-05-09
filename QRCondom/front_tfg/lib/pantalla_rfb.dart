import 'package:flutter_rfb/flutter_rfb.dart';
import 'package:flutter/material.dart';
import 'package:android_id/android_id.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';

class PantallaRfb extends StatelessWidget {
  final int port;

  const PantallaRfb({super.key, required this.port});

  void cerrarSesion(BuildContext context) async {
    final androidIdPlugin = AndroidId();
    final androidId = await androidIdPlugin.getId();

    var url = Uri.parse('http://192.168.1.46:80/cerrar_sesion_consulta_segura');

    var body = json.encode({
      'android_id': androidId,
    });

    try {
      var response = await http.post(
        url,
        headers: {'Content-Type': 'application/json'},
        body: body,
      );

      if (response.statusCode == 200) {
        Navigator.of(context).popUntil((route) => route.isFirst);
      } else {
        print('Error wacho: ${response.statusCode}');
      }
    } catch (e) {
      print('Request failed: $e');
    }
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      home: Scaffold(
        appBar: AppBar(title: const Text('Flutter RFB Example')),
        body: Center(
          child: Column(
            children: [
              InteractiveViewer(
                constrained: true,
                maxScale: 10,
                child: RemoteFrameBufferWidget(
                  hostName: '192.168.1.46',
                  port: 5900 + port,
                  onError: (Object error) {
                    print('Error de conexión: $error');
                  },
                ),
              ),
              const SizedBox(height: 16),
              FloatingActionButton.small(
                backgroundColor: Colors.black,
                onPressed: () => cerrarSesion(context),
                child: const Icon(Icons.close),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
