import 'package:flutter/material.dart';
import 'package:front_tfg/analisis_QR.dart';
import 'package:qr_code_scanner_plus/qr_code_scanner_plus.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';
import 'package:geolocator/geolocator.dart';
import 'package:android_id/android_id.dart';

class Escaneo_QR extends StatefulWidget {
  @override
  const Escaneo_QR({super.key});
  _Escaneo_QR_State createState() => _Escaneo_QR_State();
}

Future<Position?> sacarPosicion() async {
  Position? posicion;
  bool servicioHabilitado;
  LocationPermission permiso;

  servicioHabilitado = await Geolocator.isLocationServiceEnabled();

  print(servicioHabilitado ? "habilitado" : "no habilitado");

  !servicioHabilitado ? await Geolocator.openLocationSettings() : null;

  if (servicioHabilitado) {
    permiso = await Geolocator.checkPermission();
    if (permiso == LocationPermission.denied) {
      print("pedimos permisos");
      permiso = await Geolocator.requestPermission();
      if (permiso == LocationPermission.denied) {
      } else {
        posicion = await Geolocator.getCurrentPosition(
          desiredAccuracy: LocationAccuracy.high,
        );
      }
    } else {
      posicion = await Geolocator.getCurrentPosition(
        desiredAccuracy: LocationAccuracy.high,
      );
    }
  }

  return posicion;
}

class _Escaneo_QR_State extends State<Escaneo_QR> {
  final GlobalKey qrKey = GlobalKey(debugLabel: 'QR');
  QRViewController? controller;
  Barcode? result;

  @override
  void dispose() {
    controller?.dispose();
    super.dispose();
  }

  void analisis_qr(analizador, qr) {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (context) => AnalisisQr(analisis: analizador, qr: qr),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      home: Scaffold(
        appBar: AppBar(title: Text('QR Code Scanner')),
        body: Column(
          children: [
            Expanded(
              flex: 5,
              child: QRView(key: qrKey, onQRViewCreated: _onQRViewCreated),
            ),
            Expanded(
              flex: 1,
              child: Center(
                child:
                    (result != null)
                        ? Text('Data: ${result!.code}')
                        : Text('Scan a code'),
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _onQRViewCreated(QRViewController controller) {
    this.controller = controller;

    controller.scannedDataStream.listen((scanData) async {
      controller.pauseCamera();

      var url = Uri.parse('http://192.168.1.46:80/analisis_qr');
      final Position? posicion = await sacarPosicion();
      final androidIdPlugin = AndroidId();
      final androidId = await androidIdPlugin.getId();

      var body = json.encode({
        'code_qr': scanData.code,
        'Localizacion':
            (posicion == null)
                ? ''
                : {
                  'latitude': posicion.latitude,
                  'longitude': posicion.longitude,
                },
        'androidId': androidId,
      });

      try {
        print("Puta estamos en el try");
        var response = await http.post(
          url,
          headers: {'Content-Type': 'application/json'},
          body: body,
        );

        print("Se lanzo el post");

        if (response.statusCode == 201) {
          setState(() {
            analisis_qr(response.body, scanData.code);
          });
        } else {
          print('Error wacho: ${response.statusCode}');
          controller.resumeCamera();
        }

        //var data = json.decode(response.body);

        //print(response.body);
      } catch (e) {
        controller.resumeCamera();
        print('Request failed: $e'); // Si hay un error en la red, lo mostramos
      }
    });

    //completar
  }
}
