import 'package:flutter_rfb/flutter_rfb.dart';
import 'package:flutter/material.dart';
import 'package:android_id/android_id.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';
import 'dart:async';

class PantallaRfb extends StatefulWidget {
  final int port;

  const PantallaRfb({super.key, required this.port});

  @override
  _PantallaRfb createState() => _PantallaRfb();
}

class _PantallaRfb extends State<PantallaRfb> {
  Key keyIteractiveViewer = const ValueKey('init_iteractive');
  int segundos = 20;
  bool cargado = false;

  @override
  void initState() {
    super.initState();
    _temporizador();
  }

  void _temporizador() {
    Timer.periodic(const Duration(seconds: 1), (timer) {
      if (segundos > 1) {
        setState(() {
          segundos--;
        });
      } else {
        timer.cancel();
        setState(() {
          cargado = true;
        });
      }
    });
  }

  void cerrarSesion(BuildContext context) async {
    final androidIdPlugin = AndroidId();
    final androidId = await androidIdPlugin.getId();

    var url = Uri.parse('http://192.168.1.46:80/cerrar_sesion_consulta_segura');

    var body = json.encode({'android_id': androidId});

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
        backgroundColor: const Color(0xFFE3F2FD),
        appBar: AppBar(
          title: Text('Consulta Segura', style: TextStyle(color: Colors.white)),
          backgroundColor: const Color(0xFF1565C0),
          iconTheme: const IconThemeData(color: Colors.white),
        ),
        body: Stack(
          children: [
            Column(
              children: [
                Container(
                  margin: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(16),
                    boxShadow: [
                      BoxShadow(
                        color: Colors.blue.withOpacity(0.1),
                        blurRadius: 10,
                        offset: const Offset(0, 4),
                      ),
                    ],
                  ),
                  child:
                      cargado
                          ? UsuarioVNC(widget.port, '192.168.1.46')
                          : TextoCargando(segundos),
                  /*Text(
                            "Cargando... espera $segundos segundos",
                            style: const TextStyle(fontSize: 20),
                          ),*/
                ),
              ],
            ),
            Align(
              alignment: Alignment.bottomRight,
              child: Padding(
                padding: const EdgeInsets.all(16.0),
                child: FloatingActionButton(
                  backgroundColor: const Color(0xFF0D47A1), // Azul oscuro
                  onPressed: () => cerrarSesion(context),
                  child: const Icon(Icons.close, color: Colors.white),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class UsuarioVNC extends StatelessWidget {
  final int puerto;
  final String ip;

  UsuarioVNC(this.puerto, this.ip);

  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(16),
      child: InteractiveViewer(
        constrained: true,
        maxScale: 10,
        child: RemoteFrameBufferWidget(
          hostName: ip,
          port: 5900 + puerto,
          onError: (Object error) {
            print(puerto);
            print('Error de conexión: $error');
          },
        ),
      ),
    );
  }
}

class TextoCargando extends StatelessWidget {
  int segundos;

  TextoCargando(this.segundos);

  Widget build(BuildContext context) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Spacer(),
        const SizedBox(
          width: 24,
          height: 24,
          child: CircularProgressIndicator(
            strokeWidth: 2,
            valueColor: AlwaysStoppedAnimation<Color>(Color(0xFF1565C0)),
          ),
        ),
        const SizedBox(width: 16),
        Text(
          "Cargando... espera $segundos segundos",
          style: const TextStyle(
            fontSize: 18,
            fontWeight: FontWeight.w500,
            color: Color(0xFF0D47A1),
          ),
        ),
        const Spacer(),
      ],
    );
  }
}
