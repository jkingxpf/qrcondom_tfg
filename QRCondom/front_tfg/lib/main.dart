import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:front_tfg/pantalla_QR.dart';
import 'package:front_tfg/pantalla_rfb.dart';
import 'package:device_info_plus/device_info_plus.dart';
import 'package:android_id/android_id.dart';
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
  };

  return jsonEncode(datosFiltrados);
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  // This widget is the root of your application.
  @override
  Widget build(BuildContext context) {
    return MaterialApp(title: 'Front_TFG', home: MyHomePage('Titulo TFG?'));
  }
}

class MyHomePage extends StatelessWidget {
  final String title;
  MyHomePage(this.title);

  @override
  Widget build(BuildContext context) {
    double anchoScreen = MediaQuery.of(context).size.width;
    double largoScreen = MediaQuery.of(context).size.height;

    void enviarDatos() async {
      var url = Uri.parse('http://192.168.1.46:80/guardar_disp');
      try {
        print("Puta estamos en el try envio datos");
        
        final bodyJson = await infoDispJson();
        
        print(bodyJson);

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

        print(response.body);
      } catch (e) {
        print('Request failed: $e'); // Si hay un error en la red, lo mostramos
      }
    }

    return Scaffold(
      appBar: AppBar(
        title: Text(
          title,
          style: TextStyle(color: const Color.fromARGB(172, 247, 252, 245)),
        ),
        backgroundColor: const Color.fromARGB(255, 0, 77, 64), // Verde oscuro
      ),
      body: Container(
        color: Colors.black, // Fondo negro para sensación de solidez
        width: anchoScreen,
        height: largoScreen,
        child: Column(
          mainAxisAlignment: MainAxisAlignment.start,
          children: [
            TextoContainer("introducción texto"),
            ElevatedButton(
              onPressed: enviarDatos,
              style: ElevatedButton.styleFrom(
                backgroundColor: const Color.fromARGB(255, 0, 150, 136),
                foregroundColor: Colors.white, // Texto blanco para contraste
              ),
              child: Text("AAAAABOTON"),
            ),
            ElevatedButton(
              onPressed:
                  () => {},
              style: ElevatedButton.styleFrom(
                backgroundColor: const Color.fromARGB(255, 0, 150, 136),
                foregroundColor: Colors.white, // Texto blanco para contraste
              ),
              child: Text("boton rfb"),
            ),
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
      color: Colors.teal,
      padding: EdgeInsets.all(16),
      child: Text(texto, style: TextStyle(fontSize: 18, color: Colors.white)),
    );
  }
}
