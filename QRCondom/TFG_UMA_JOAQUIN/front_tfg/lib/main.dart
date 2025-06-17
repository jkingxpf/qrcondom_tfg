import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:device_info_plus/device_info_plus.dart';
import 'package:android_id/android_id.dart';
import 'package:front_tfg/pantalla_QR.dart';
import 'package:http/http.dart' as http;

void main() {
  runApp(const MyApp());
}

Future<String> infoDispJson() async {
  final infoDispositivo = DeviceInfoPlugin();
  final androidInfo = await infoDispositivo.androidInfo;
  final androidIdPlugin = AndroidId();
  final androidId = await androidIdPlugin.getId();

  final datosFiltrados = {
    'Dispositivo': {
      'androidId': androidId,
      'version.securityPatch': androidInfo.version.securityPatch,
      'version.sdkInt': androidInfo.version.sdkInt,
      'version.release': androidInfo.version.release,
      'version.previewSdkInt': androidInfo.version.previewSdkInt,
      'version.incremental': androidInfo.version.incremental,
      'version.codename': androidInfo.version.codename,
      'version.baseOS': androidInfo.version.baseOS,
      'board': androidInfo.board,
      'bootloader': androidInfo.bootloader,
      'brand': androidInfo.brand,
      'device': androidInfo.device,
      'display': androidInfo.display,
      'fingerprint': androidInfo.fingerprint,
      'hardware': androidInfo.hardware,
      'host': androidInfo.host,
      'id': androidInfo.id,
      'manufacturer': androidInfo.manufacturer,
      'model': androidInfo.model,
      'product': androidInfo.product,
      'supported32BitAbis': androidInfo.supported32BitAbis,
      'supported64BitAbis': androidInfo.supported64BitAbis,
      'type': androidInfo.type,
      'isPhysicalDevice': androidInfo.isPhysicalDevice,
      'systemFeatures': androidInfo.systemFeatures,
      'serialNumber': androidInfo.serialNumber,
      'isLowRamDevice': androidInfo.isLowRamDevice,
    },
  };

  print(datosFiltrados);

  return jsonEncode(datosFiltrados);
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'QRCondomTFG',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.lightBlue),
        useMaterial3: true,
      ),
      home: MyHomePage('QRCondom'),
    );
  }
}

class MyHomePage extends StatelessWidget {
  final String title;
  MyHomePage(this.title);

  @override
  Widget build(BuildContext context) {
    double anchoScreen = MediaQuery.of(context).size.width;
    double largoScreen = MediaQuery.of(context).size.height;

    String texto =
        "Este proyecto recopilará información de su dispositivo con el fin de gestionar correctamente los procesos internos y analizar patrones asociados a códigos QR potencialmente maliciosos. Si está de acuerdo con estas condiciones, pulse el botón para continuar con el análisis.";

    return Scaffold(
      appBar: AppBar(
        title: Text(title, style: const TextStyle(color: Colors.white)),
        backgroundColor: const Color(0xFF1565C0),
        centerTitle: true,
      ),
      body: Container(
        color: const Color(0xFFE3F2FD),
        width: anchoScreen,
        height: largoScreen,
        child: Column(
          children: [
            TextoContainer(texto),
            const Spacer(),
            const Spacer(),
            const Spacer(),
            const Spacer(),

            BotonBajoCentro("Analizador"),
            const Spacer(),
          ],
        ),
      ),
    );
  }
}

class TextoContainer extends StatelessWidget {
  final String texto;
  TextoContainer(this.texto);

  @override
  Widget build(BuildContext context) {
    return Container(
      color: const Color(0xFFE3F2FD),
      padding: const EdgeInsets.all(20),
      child: Text(
        texto,
        style: const TextStyle(fontSize: 16, color: Color(0xFF0D47A1)),
        textAlign: TextAlign.justify,
      ),
    );
  }
}

class BotonBajoCentro extends StatelessWidget {
  final String nombreBoton;
  BotonBajoCentro(this.nombreBoton);

  @override
  Widget build(BuildContext context) {
    void enviarDatos() async {
      var url = Uri.parse('http://192.168.1.46:80/guardar_disp');
      try {
        final bodyJson = await infoDispJson();

        var response = await http.post(
          url,
          headers: {'Content-Type': 'application/json'},
          body: bodyJson,
        );

        if (response.statusCode == 200) {
          Navigator.push(
            context,
            MaterialPageRoute(builder: (context) => Escaneo_QR()),
          );
        }
      } catch (e) {
        print('Error try: $e');
      }
    }

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 24.0),
      child: Center(
        child: ElevatedButton(
          key: Key("boton_consentimiento"),
          onPressed: enviarDatos,
          style: ElevatedButton.styleFrom(
            backgroundColor: const Color(0xFF1976D2),
            padding: const EdgeInsets.symmetric(horizontal: 40, vertical: 15),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(12),
            ),
          ),
          child: Text(
            nombreBoton,
            style: const TextStyle(fontSize: 16, color: Colors.white),
          ),
        ),
      ),
    );
  }
}
