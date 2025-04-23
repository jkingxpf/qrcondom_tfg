import 'package:flutter_rfb/flutter_rfb.dart';
import 'package:flutter/material.dart';

class PantallaRfb extends StatelessWidget {
  final int port;

  const PantallaRfb({super.key,required this.port});

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
                  //password: 'password',
                ),
              ),
              Positioned(
                bottom: 16,
                left: 16,
                child: FloatingActionButton.small(
                  backgroundColor: Colors.black,
                  onPressed: () => (),
                  child: const Icon(Icons.close),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
